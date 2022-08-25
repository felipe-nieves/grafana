import React, { useMemo } from 'react';

import { selectors } from '@grafana/e2e-selectors';
import { config } from '@grafana/runtime';
import { Button, Field, Form, HorizontalGroup, Input, LinkButton } from '@grafana/ui';
import { DashboardPicker } from 'app/core/components/Select/DashboardPicker';

import { TagFilter } from '../../core/components/TagFilter/TagFilter';
import { getGrafanaSearcher } from '../search/service';

import { PlaylistTable } from './PlaylistTable';
import { Playlist } from './types';
import { usePlaylistItems } from './usePlaylistItems';

interface PlaylistFormProps {
  onSubmit: (playlist: Playlist) => void;
  playlist: Playlist;
}

export const PlaylistForm = ({ onSubmit, playlist }: PlaylistFormProps) => {
  const { name, interval, items: propItems } = playlist;
  const tagOptions = useMemo(() => {
    return () => getGrafanaSearcher().tags({ kind: ['dashboard'] });
  }, []);

  const { items, addById, addByTag, deleteItem, moveDown, moveUp } = usePlaylistItems(propItems);

  return (
    <div>
      <Form onSubmit={(list: Playlist) => onSubmit({ ...list, items })} validateOn={'onBlur'}>
        {({ register, errors }) => {
          const isDisabled = items.length === 0 || Object.keys(errors).length > 0;
          return (
            <>
              <Field label="Name" invalid={!!errors.name} error={errors?.name?.message}>
                <Input
                  type="text"
                  {...register('name', { required: 'Name is required' })}
                  placeholder="Name"
                  defaultValue={name}
                  aria-label={selectors.pages.PlaylistForm.name}
                />
              </Field>
              <Field label="Interval" invalid={!!errors.interval} error={errors?.interval?.message}>
                <Input
                  type="text"
                  {...register('interval', { required: 'Interval is required' })}
                  placeholder="5m"
                  defaultValue={interval ?? '5m'}
                  aria-label={selectors.pages.PlaylistForm.interval}
                />
              </Field>

              <PlaylistTable items={items} onMoveUp={moveUp} onMoveDown={moveDown} onDelete={deleteItem} />

              <div className="gf-form-group">
                <h3 className="page-headering">Add dashboards</h3>

                <Field label="Add by title">
                  <DashboardPicker
                    key={items.length}
                    onChange={addById}
                    id="dashboard-picker"
                    placeholder="search for dashboard"
                  />
                </Field>

                <Field label="Add by tag">
                  <TagFilter
                    isClearable
                    tags={[]}
                    hideValues
                    tagOptions={tagOptions}
                    onChange={addByTag}
                    placeholder={''}
                  />
                </Field>
              </div>

              <HorizontalGroup>
                <Button type="submit" variant="primary" disabled={isDisabled}>
                  Save
                </Button>
                <LinkButton variant="secondary" href={`${config.appSubUrl}/playlists`}>
                  Cancel
                </LinkButton>
              </HorizontalGroup>
            </>
          );
        }}
      </Form>
    </div>
  );
};
