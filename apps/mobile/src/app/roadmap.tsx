import { View } from 'react-native';

import { Headline } from '@/components/Headline';
import { RoadmapGraph } from '@/components/roadmap/RoadmapGraph';
import { RoadmapList } from '@/components/roadmap/RoadmapList';
import { ErrorState, LoadingState, SectionHeader, SegmentedPill, StatusChip } from '@/components/ui/Bits';
import { Card } from '@/components/ui/Card';
import { DotPattern } from '@/components/ui/Decor';
import { Screen, useLayout } from '@/components/ui/Screen';
import { useRoadmap } from '@/lib/queries';
import { useUI } from '@/lib/store';

export default function RoadmapScreen() {
  const { data, isPending, isError, error, refetch, isRefetching } = useRoadmap();
  const { wide, contentWidth } = useLayout();
  const view = useUI((s) => s.roadmapView);
  const setView = useUI((s) => s.setRoadmapView);
  const pad = wide ? 32 : 16;

  return (
    <Screen onRefresh={refetch} refreshing={isRefetching}>
      <SectionHeader
        wide={wide}
        label="Roadmap"
        title={<Headline size={wide ? 60 : 36} before="Every skill from A0 to" gradient="C2" after=", in order." />}
        description="Each skill unlocks once its prerequisites are mastered. B2 is your goal; C1 and C2 are optional advanced tracks."
        aside={
          <SegmentedPill
            value={view}
            onChange={setView}
            options={[
              { value: 'graph', label: 'Graph', icon: 'graph' },
              { value: 'list', label: 'List', icon: 'list' },
            ]}
          />
        }
      />
      {isPending ? (
        <LoadingState label="Loading roadmap" />
      ) : isError ? (
        <ErrorState title="Can't load the roadmap" message={error.message} onRetry={refetch} />
      ) : (
        <>
          <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 6, marginBottom: 16 }} accessibilityLabel="Legend">
            {(['mastered', 'in_progress', 'available', 'locked', 'coming_soon'] as const).map((s) => (
              <StatusChip key={s} status={s} />
            ))}
          </View>
          {view === 'graph' ? (
            <Card padding={pad} spotlight={false} decoration={<DotPattern size={16} r={1.2} color="rgba(17,17,19,0.10)" />}>
              <RoadmapGraph nodes={data.nodes} levels={data.summary.levels} width={contentWidth - pad * 2 - 2} wide={wide} nextNodeId={data.nextNodeId} />
            </Card>
          ) : (
            <RoadmapList nodes={data.nodes} levels={data.summary.levels} />
          )}
        </>
      )}
    </Screen>
  );
}
