import React from 'react';
import { GraphiteQueryEditorState } from '../state/store';
import { TagsSection } from './TagsSection';
import { MetricsSection } from './MetricsSection';
import { Section } from '@grafana/ui';

type Props = {
  state: GraphiteQueryEditorState;
};

export function SeriesSection({ state }: Props) {
  const sectionContent = state.queryModel?.seriesByTagUsed ? (
    <TagsSection tags={state.queryModel?.tags} state={state} />
  ) : (
    <MetricsSection segments={state.segments} state={state} />
  );

  return (
    <Section label="Series" fill={true}>
      {sectionContent}
    </Section>
  );
}
