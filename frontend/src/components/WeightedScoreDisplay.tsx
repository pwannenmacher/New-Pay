import { useEffect, useState } from 'react';
import { Badge, Group, Loader, Text, Tooltip } from '@mantine/core';
import { selfAssessmentService } from '../services/selfAssessment';
import type { WeightedScore } from '../types';

function useWeightedScore(assessmentId: number) {
  const [score, setScore] = useState<WeightedScore | null>(null);
  const [loading, setLoading] = useState(true);

  // Reset to loading state when the assessment changes (instead of inside the effect)
  const [prevAssessmentId, setPrevAssessmentId] = useState(assessmentId);
  if (assessmentId !== prevAssessmentId) {
    setPrevAssessmentId(assessmentId);
    setLoading(true);
  }

  useEffect(() => {
    let ignore = false;
    selfAssessmentService
      .getWeightedScore(assessmentId)
      .then((data) => {
        if (!ignore) setScore(data);
      })
      .catch((error) => {
        console.error('Error loading weighted score:', error);
        if (!ignore) setScore(null);
      })
      .finally(() => {
        if (!ignore) setLoading(false);
      });
    return () => {
      ignore = true;
    };
  }, [assessmentId]);

  return { score, loading };
}

interface WeightedScoreBadgeProps {
  assessmentId: number;
  compact?: boolean;
}

export function WeightedScoreBadge({ assessmentId, compact = false }: WeightedScoreBadgeProps) {
  const { score, loading } = useWeightedScore(assessmentId);

  if (loading) {
    return <Loader size="xs" />;
  }

  if (!score || !score.is_complete) {
    return null;
  }

  const displayText = compact
    ? `${score.overall_level} (${score.weighted_average.toFixed(2)})`
    : `Gesamtlevel: ${score.overall_level} (Ø ${score.weighted_average.toFixed(2)})`;

  return (
    <Tooltip label={`Gewichteter Durchschnitt: ${score.weighted_average.toFixed(4)}`}>
      <Badge size={compact ? 'sm' : 'lg'} variant="light" color="teal">
        {displayText}
      </Badge>
    </Tooltip>
  );
}

export function WeightedScoreDisplay({ assessmentId }: { assessmentId: number }) {
  const { score, loading } = useWeightedScore(assessmentId);

  if (loading) {
    return (
      <Group>
        <Loader size="sm" />
        <Text size="sm" c="dimmed">
          Berechne Gesamtlevel...
        </Text>
      </Group>
    );
  }

  if (!score) {
    return null;
  }

  if (!score.is_complete) {
    return (
      <Text size="sm" c="dimmed">
        Gesamtlevel wird berechnet, sobald alle Kategorien ausgefüllt sind.
      </Text>
    );
  }

  return (
    <Group>
      <Text size="sm" fw={500}>
        Gesamtlevel:
      </Text>
      <Badge size="lg" variant="filled" color="teal">
        {score.overall_level}
      </Badge>
      <Text size="sm" c="dimmed">
        (gewichteter Durchschnitt: {score.weighted_average.toFixed(2)})
      </Text>
    </Group>
  );
}
