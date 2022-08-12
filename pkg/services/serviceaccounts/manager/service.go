package manager

import (
	"context"
	"fmt"
	"time"

	"github.com/grafana/grafana/pkg/api/routing"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/infra/usagestats"
	"github.com/grafana/grafana/pkg/services/accesscontrol"
	"github.com/grafana/grafana/pkg/services/serviceaccounts"
	"github.com/grafana/grafana/pkg/services/serviceaccounts/api"
	"github.com/grafana/grafana/pkg/services/serviceaccounts/database"
	"github.com/grafana/grafana/pkg/services/serviceaccounts/toucan"
	"github.com/grafana/grafana/pkg/setting"
)

const (
	metricsCollectionInterval      = time.Minute * 30
	defaultTokenCollectionInterval = time.Minute * 5
)

type ServiceAccountsService struct {
	store         serviceaccounts.Store
	log           log.Logger
	backgroundLog log.Logger
	toucanService toucan.Checker

	checkTokenLeaks    bool
	checkTokenInterval time.Duration
}

func ProvideServiceAccountsService(
	cfg *setting.Cfg,
	ac accesscontrol.AccessControl,
	routeRegister routing.RouteRegister,
	usageStats usagestats.Service,
	serviceAccountsStore serviceaccounts.Store,
	permissionService accesscontrol.ServiceAccountPermissionsService,
) (*ServiceAccountsService, error) {
	database.InitMetrics()
	s := &ServiceAccountsService{
		store:         serviceAccountsStore,
		log:           log.New("serviceaccounts"),
		backgroundLog: log.New("serviceaccounts.background"),
	}

	if err := RegisterRoles(ac); err != nil {
		s.log.Error("Failed to register roles", "error", err)
	}

	usageStats.RegisterMetricsFunc(s.store.GetUsageMetrics)

	serviceaccountsAPI := api.NewServiceAccountsAPI(cfg, s, ac, routeRegister, s.store, permissionService)
	serviceaccountsAPI.RegisterAPIEndpoints()

	s.checkTokenLeaks = cfg.SectionWithEnvOverrides("toucan").Key("enabled").MustBool(false)
	if s.checkTokenLeaks {
		s.checkTokenInterval = cfg.SectionWithEnvOverrides("toucan").
			Key("interval").MustDuration(defaultTokenCollectionInterval)

		// Enforce a minimum interval of 1 minute.
		if s.checkTokenInterval < time.Minute {
			s.backgroundLog.Warn("token leak check interval is too low, increasing to " +
				defaultTokenCollectionInterval.String())

			s.checkTokenInterval = defaultTokenCollectionInterval
		}

		s.toucanService = toucan.NewService(&toucan.MockTokenRetriever{})
	}

	return s, nil
}

func (sa *ServiceAccountsService) Run(ctx context.Context) error {
	sa.backgroundLog.Debug("service initialized")

	if _, err := sa.store.GetUsageMetrics(ctx); err != nil {
		sa.log.Warn("Failed to get usage metrics", "error", err.Error())
	}

	updateStatsTicker := time.NewTicker(metricsCollectionInterval)
	defer updateStatsTicker.Stop()

	tokenCheckTicker := time.NewTicker(defaultTokenCollectionInterval)

	if !sa.checkTokenLeaks {
		tokenCheckTicker.Stop()
	} else {
		sa.backgroundLog.Debug("enabled token leak check")

		defer tokenCheckTicker.Stop()
	}

	for {
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("context error in service account background service: %w", ctx.Err())
			}

			sa.backgroundLog.Debug("stopped service account background service")

			return nil
		case <-updateStatsTicker.C:
			sa.backgroundLog.Debug("updating usage metrics")

			if _, err := sa.store.GetUsageMetrics(ctx); err != nil {
				sa.backgroundLog.Warn("Failed to get usage metrics", "error", err.Error())
			}
		case <-tokenCheckTicker.C:
			sa.backgroundLog.Debug("checking for leaked tokens")

			if err := sa.toucanService.CheckTokens(ctx); err != nil {
				sa.backgroundLog.Warn("Failed to check for leaked tokens", "error", err.Error())
			}
		}
	}
}

func (sa *ServiceAccountsService) CreateServiceAccount(ctx context.Context, orgID int64, saForm *serviceaccounts.CreateServiceAccountForm) (*serviceaccounts.ServiceAccountDTO, error) {
	return sa.store.CreateServiceAccount(ctx, orgID, saForm)
}

func (sa *ServiceAccountsService) DeleteServiceAccount(ctx context.Context, orgID, serviceAccountID int64) error {
	return sa.store.DeleteServiceAccount(ctx, orgID, serviceAccountID)
}

func (sa *ServiceAccountsService) RetrieveServiceAccountIdByName(ctx context.Context, orgID int64, name string) (int64, error) {
	return sa.store.RetrieveServiceAccountIdByName(ctx, orgID, name)
}
