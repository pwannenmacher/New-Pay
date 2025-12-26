import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  Container,
  Paper,
  Title,
  Text,
  Button,
  Grid,
  Stack,
  Badge,
  Group,
  Table,
  Alert,
  Divider,
  Card,
  LoadingOverlay,
} from '@mantine/core';
import {
  IconArrowLeft,
  IconInfoCircle,
  IconMessageCircle,
  IconUserCheck,
  IconCheck,
} from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import discussionService, { type DiscussionResult } from '../services/discussion';
import { useAuth } from '../contexts/AuthContext';

export function UserDiscussionPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { user } = useAuth();
  const assessmentId = parseInt(id || '0');

  const [loading, setLoading] = useState(true);
  const [data, setData] = useState<DiscussionResult | null>(null);
  const [confirming, setConfirming] = useState(false);

  useEffect(() => {
    loadData();
  }, [assessmentId]);

  const loadData = async () => {
    try {
      setLoading(true);
      const discussionData = await discussionService.getDiscussionResult(assessmentId);
      setData(discussionData);
    } catch (error: any) {
      console.error('Error loading discussion data:', error);
      notifications.show({
        title: 'Fehler',
        message: error.response?.data?.error || 'Diskussionsdaten konnten nicht geladen werden',
        color: 'red',
      });
    } finally {
      setLoading(false);
    }
  };

  const handleConfirmMeeting = async () => {
    try {
      setConfirming(true);
      await discussionService.createConfirmation(assessmentId, 'owner');
      notifications.show({
        title: 'Erfolg',
        message: 'Besprechung bestätigt',
        color: 'green',
      });
      await loadData();
    } catch (error: any) {
      console.error('Error confirming meeting:', error);
      notifications.show({
        title: 'Fehler',
        message: error.response?.data?.error || 'Bestätigung konnte nicht gespeichert werden',
        color: 'red',
      });
    } finally {
      setConfirming(false);
    }
  };

  // Check if reviewer has confirmed
  const reviewerConfirmed = data?.confirmations?.some((c) => c.user_type === 'reviewer');
  
  // Check if current user (owner) has confirmed
  const currentUserConfirmed = data?.confirmations?.some(
    (c) => c.user_id === user?.id && c.user_type === 'owner'
  );

  if (loading) {
    return (
      <Container size="xl" py="xl">
        <LoadingOverlay visible />
      </Container>
    );
  }

  if (!data) {
    return (
      <Container size="xl" py="xl">
        <Alert color="red" title="Fehler">
          Diskussionsdaten konnten nicht geladen werden
        </Alert>
      </Container>
    );
  }

  return (
    <Container size="xl" py="xl">
      <Stack gap="lg">
        {/* Header */}
        <Group justify="space-between">
          <div>
            <Group gap="xs" mb="xs">
              <Button
                variant="subtle"
                leftSection={<IconArrowLeft size={16} />}
                onClick={() => navigate('/self-assessments')}
              >
                Zurück zu Meine Selbsteinschätzungen
              </Button>
            </Group>
            <Title order={2}>Ergebnis der Besprechung</Title>
            <Text size="sm" c="dimmed">
              Assessment ID: {assessmentId}
            </Text>
          </div>
          <Badge color="violet" size="lg" leftSection={<IconMessageCircle size={16} />}>
            Besprechung
          </Badge>
        </Group>

        {/* Info Alert */}
        <Alert icon={<IconInfoCircle size={16} />} color="blue">
          Hier sehen Sie das Ergebnis Ihrer Selbsteinschätzung im Vergleich mit der Bewertung des
          Reviewer-Gremiums. Diese Ansicht zeigt die finale Bewertung nach der Besprechung.
        </Alert>

        {/* Comparison Table */}
        <Paper withBorder p="md">
          <Title order={3} mb="md">
            Vergleich der Bewertungen
          </Title>
          <Table striped highlightOnHover>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Kategorie</Table.Th>
                <Table.Th>Ihre Selbsteinschätzung</Table.Th>
                <Table.Th>Reviewer-Bewertung</Table.Th>
                <Table.Th>Begründung</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {data.category_results?.map((categoryResult) => (
                <Table.Tr key={categoryResult.id}>
                  <Table.Td>
                    <Text fw={600}>{categoryResult.category_name}</Text>
                  </Table.Td>
                  <Table.Td>
                    {categoryResult.user_level_id ? (
                      <Badge size="sm" color="blue">
                        {categoryResult.user_level_name}
                      </Badge>
                    ) : (
                      <Text size="sm" c="dimmed">
                        Keine Angabe
                      </Text>
                    )}
                  </Table.Td>
                  <Table.Td>
                    <Badge size="sm" color="green">
                      {categoryResult.reviewer_level_name}
                    </Badge>
                  </Table.Td>
                  <Table.Td>
                    {categoryResult.justification ? (
                      <Text size="sm" style={{ whiteSpace: 'pre-wrap' }}>
                        {categoryResult.justification}
                      </Text>
                    ) : (
                      <Text size="sm" c="dimmed">
                        Keine Begründung
                      </Text>
                    )}
                  </Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        </Paper>

        {/* Final Consolidation Result */}
        <Card withBorder shadow="sm" p="md">
          <Title order={4} mb="md">
            Gesamtbewertung
          </Title>

          <Grid>
            <Grid.Col span={4}>
              <Stack gap="xs">
                <Text size="sm" fw={500} c="dimmed">
                  Gewichtetes Gesamtergebnis
                </Text>
                <div>
                  <Badge size="xl" color="green" variant="filled">
                    {data.weighted_overall_level_name}
                  </Badge>
                  <Text size="xs" c="dimmed" mt="xs">
                    (Durchschnitt: {data.weighted_overall_level_number.toFixed(2)})
                  </Text>
                </div>
              </Stack>
            </Grid.Col>

            <Grid.Col span={8}>
              <Stack gap="xs">
                <Text size="sm" fw={500} c="dimmed">
                  Abschlusskommentar des Reviewer-Gremiums
                </Text>
                <Text size="sm" style={{ whiteSpace: 'pre-wrap' }}>
                  {data.final_comment || 'Kein Kommentar verfügbar'}
                </Text>
              </Stack>
            </Grid.Col>
          </Grid>

          <Divider my="md" />

          <Group gap="xs">
            <Text size="sm" fw={500}>
              Reviewer:
            </Text>
            {data.reviewers?.map((reviewer) => (
              <Badge key={reviewer.id} color="blue">
                {reviewer.reviewer_name}
              </Badge>
            ))}
          </Group>
        </Card>

        {/* Discussion Note (if available) */}
        {data.discussion_note && (
          <Paper withBorder p="md">
            <Title order={4} mb="md">
              Notizen zur Besprechung
            </Title>
            <Text size="sm" style={{ whiteSpace: 'pre-wrap' }}>
              {data.discussion_note}
            </Text>
            {data.user_approved_at && (
              <Text size="xs" c="dimmed" mt="md">
                Bestätigt am {new Date(data.user_approved_at).toLocaleString('de-DE')}
              </Text>
            )}
          </Paper>
        )}

        {/* Meeting Confirmation Section */}
        <Paper withBorder p="md">
          <Title order={4} mb="md">
            Besprechungsbestätigung
          </Title>

          <Alert icon={<IconInfoCircle size={16} />} color="blue" mb="md">
            Bitte bestätigen Sie, dass die Besprechung mit den Reviewern stattgefunden hat und Sie
            die Bewertung verstanden haben.
          </Alert>

          <Stack gap="md">
            <div>
              <Text size="sm" fw={500} mb="xs">
                Bestätigungen:
              </Text>
              {data.confirmations && data.confirmations.length > 0 ? (
                <Stack gap="xs">
                  {data.confirmations.map((confirmation) => (
                    <Group key={confirmation.id}>
                      <Badge
                        color={confirmation.user_type === 'reviewer' ? 'blue' : 'green'}
                        leftSection={<IconUserCheck size={14} />}
                      >
                        {confirmation.user_type === 'reviewer' ? 'Reviewer' : 'Sie'}
                      </Badge>
                      <Text size="sm">
                        {confirmation.user_type === 'reviewer' && `${confirmation.user_name || 'Unbekannt'} - `}
                        {new Date(confirmation.confirmed_at).toLocaleString('de-DE')}
                      </Text>
                    </Group>
                  ))}
                </Stack>
              ) : (
                <Text size="sm" c="dimmed">
                  Noch keine Bestätigungen
                </Text>
              )}
            </div>

            {!reviewerConfirmed && (
              <Alert color="orange" icon={<IconInfoCircle size={16} />}>
                Die Reviewer müssen die Besprechung zuerst bestätigen, bevor Sie bestätigen können.
              </Alert>
            )}

            {reviewerConfirmed && !currentUserConfirmed && (
              <Button
                onClick={handleConfirmMeeting}
                loading={confirming}
                leftSection={<IconUserCheck size={16} />}
                color="green"
              >
                Ich bestätige, dass die Besprechung stattgefunden hat
              </Button>
            )}

            {currentUserConfirmed && (
              <Alert color="green" icon={<IconCheck size={16} />}>
                Sie haben die Besprechung bereits bestätigt. Vielen Dank!
              </Alert>
            )}
          </Stack>
        </Paper>
      </Stack>
    </Container>
  );
}
