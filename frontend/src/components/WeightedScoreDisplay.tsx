import { useEffect, useState } from 'react';
import { Badge, Group, Loader, Text, Tooltip } from '@mantine/core';
import { selfAssessmentService } from '../services/selfAssessment';
import type { WeightedScore } from '../types';

interface WeightedScoreBadgeProps {
  assessmentId: number;
  compact?: boolean;
}

export function WeightedScoreBadge({ assessmentId, compact = false }: WeightedScoreBadgeProps) {
  const [score, setScore] = useState<WeightedScore | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadScore();
  }, [assessmentId]);

  const loadScore = async () => {
    try {
      setLoading(true);
      const data = await selfAssessmentService.getWeightedScore(assessmentId);
      setScore(data);
    } catch (error) {
      console.error('Error loading weighted score:', error);
      setScore(null);
    } finally {
      setLoading(false);
    }
  };

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
  const [score, setScore] = useState<WeightedScore | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadScore();
  }, [assessmentId]);

  const loadScore = async () => {
    try {
      setLoading(true);
      const data = await selfAssessmentService.getWeightedScore(assessmentId);
      setScore(data);
    } catch (error) {
      console.error('Error loading weighted score:', error);
      setScore(null);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <Group>
        <Loader size="sm" />
        <Text size="sm" c="dimmed">Berechne Gesamtlevel...</Text>
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
      <Text size="sm" fw={500}>Gesamtlevel:</Text>
      <Badge size="lg" variant="filled" color="teal">
        {score.overall_level}
      </Badge>
      <Text size="sm" c="dimmed">
        (gewichteter Durchschnitt: {score.weighted_average.toFixed(2)})
      </Text>
    </Group>
  );
}
