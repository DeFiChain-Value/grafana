import { useMemo } from 'react';
import { AlertmanagerAlert, AlertmanagerGroup } from 'app/plugins/datasource/alertmanager/types';
import { Labels } from '@grafana/data';

export const useGroupedAlerts = (alerts: AlertmanagerAlert[], groupBy: string[]) => {
  return useMemo(() => {
    return alerts.reduce((groupings, alert) => {
      const alertContainsGroupings = groupBy.every((groupByLabel) => Object.keys(alert.labels).includes(groupByLabel));

      if (alertContainsGroupings) {
        const existingGrouping = groupings.find((group) => {
          return groupBy.every((groupKey) => {
            return group.labels[groupKey] === alert.labels[groupKey];
          });
        });
        if (!existingGrouping) {
          const labels = groupBy.reduce((acc, key) => {
            acc = { ...acc, [key]: alert.labels[key] };
            return acc;
          }, {} as Labels);
          groupings.push({
            alerts: [alert],
            labels,
            receiver: {
              name: 'NONE',
            },
          });
        } else {
          existingGrouping.alerts.push(alert);
        }
      } else {
        const noGroupingGroup = groupings.find((group) => Object.keys(group.labels).length === 0);
        if (!noGroupingGroup) {
          groupings.push({ alerts: [alert], labels: {}, receiver: { name: 'NONE' } });
        } else {
          noGroupingGroup.alerts.push(alert);
        }
      }

      return groupings;
    }, [] as AlertmanagerGroup[]);
  }, [alerts, groupBy]);
};
