import { useLocalSearchParams, useRouter } from 'expo-router';

import { NodeDetailView } from '@/components/roadmap/NodeDetail';
import { ErrorState, LoadingState } from '@/components/ui/Bits';
import { Screen, useLayout } from '@/components/ui/Screen';
import { useT } from '@/i18n';
import { useErrorText } from '@/lib/errors';
import { useNode, useRoadmap } from '@/lib/queries';

export default function NodeScreen() {
  const { course, node: nodeId } = useLocalSearchParams<{ course: string; node: string }>();
  const router = useRouter();
  const { data: node, isPending, isError, error, refetch } = useNode(course, nodeId);
  // Usually cached from the course page; only used for the goal's label.
  const goal = useRoadmap(course).data?.profile.goal;
  const { wide, gap, contentWidth } = useLayout();
  const t = useT();
  const errorText = useErrorText();

  if (isPending) {
    return (
      <Screen>
        <LoadingState label={t('node.loading')} />
      </Screen>
    );
  }
  if (isError) {
    return (
      <Screen>
        <ErrorState title={'code' in error && error.code === 'node_not_found' ? t('node.notFound') : t('node.loadError')} message={errorText(error)} onRetry={refetch} />
      </Screen>
    );
  }

  return (
    <Screen>
      <NodeDetailView
        node={node}
        width={contentWidth}
        wide={wide}
        gap={gap}
        goal={goal?.label}
        onCourse={() => router.push(`/courses/${node.course.slug}`)}
        onNode={(c, id) => router.push(`/courses/${c}/${id}`)}
        onGo={(path) => router.push(path as never)}
      />
    </Screen>
  );
}
