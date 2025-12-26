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
  Textarea,
  Table,
  Alert,
  Divider,
  Card,
  LoadingOverlay,
  Checkbox,
} from '@mantine/core';
import {
  IconArrowLeft,
  IconCheck,
  IconInfoCircle,
  IconDeviceFloppy,
  IconMessageCircle,
  IconUserCheck,
  IconArchive,
} from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import discussionService, { type DiscussionResult } from '../../services/discussion';
import { useAuth } from '../../contexts/AuthContext';

export function ReviewDiscussionPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { user } = useAuth();
  const assessmentId = parseInt(id || '0');

  const [loading, setLoading] = useState(true);
  const [data, setData] = useState<DiscussionResult | null>(null);
  const [discussionComment, setDiscussionComment] = useState('');
  const [saving, setSaving] = useState(false);
  const [confirming, setConfirming] = useState(false);
  const [archiving, setArchiving] = useState(false);

  useEffect(() => {
    loadData();
  }, [assessmentId]);

  const loadData = async () => {
    try {
      setLoading(true);
      const discussionData = await discussionService.getDiscussionResult(assessmentId);
      setData(discussionData);
      setDiscussionComment(discussionData.discussion_note || '');
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

  const handleSaveComment = async () => {
    try {
      setSaving(true);
      await discussionService.updateDiscussionNote(assessmentId, discussionComment);
      notifications.show({
        title: 'Erfolg',
        message: 'Notizen gespeichert',
        color: 'green',
      });
      await loadData(); // Reload to get updated data
    } catch (error: any) {
      console.error('Error saving comment:', error);
      notifications.show({
        title: 'Fehler',
        message: error.response?.data?.error || 'Notizen konnten nicht gespeichert werden',
        color: 'red',
      });
    } finally {
      setSaving(false);
    }
  };

  const handleConfirmMeeting = async () => {
    try {
      setConfirming(true);
      await discussionService.createConfirmation(assessmentId, 'reviewer');
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

  const handleArchiveAssessment = async () => {
    try {
      setArchiving(true);
      await discussionService.archiveAssessment(assessmentId);
      notifications.show({
        title: 'Erfolg',
        message: 'Assessment wurde archiviert',
        color: 'green',
      });
      navigate('/review/open-assessments');
    } catch (error: any) {
      console.error('Error archiving assessment:', error);
      notifications.show({
        title: 'Fehler',
        message: error.response?.data?.error || 'Archivierung fehlgeschlagen',
        color: 'red',
      });
    } finally {
      setArchiving(false);
    }
  };

  // Check if current user has confirmed
  const currentUserConfirmed = data?.confirmations?.some(
    (c) => c.user_id === user?.id && c.user_type === 'reviewer'
  );

  // Check if owner has confirmed
  const ownerConfirmed = data?.confirmations?.some((c) => c.user_type === 'owner');

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
                onClick={() => navigate('/review/open-assessments')}
              >
                Zurück
              </Button>
            </Group>
            <Title order={2}>Ergebnisbesprechung</Title>
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
          Auf dieser Seite werden die Selbsteinschätzung der/des Benutzer:in mit der konsolidierten 
          Bewertung des Reviewer-Gremiums verglichen. Sie können hier Kommentare zur Diskussion hinzufügen.
        </Alert>

        {/* Comparison Table */}
        <Paper withBorder p="md">
          <Title order={3} mb="md">Vergleich der Bewertungen</Title>
          <Table striped highlightOnHover>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Kategorie</Table.Th>
                <Table.Th>Selbsteinschätzung</Table.Th>
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
                      <Text size="sm" c="dimmed">Keine Angabe</Text>
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
                      <Text size="sm" c="dimmed">Keine Begründung</Text>
                    )}
                  </Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        </Paper>

        {/* Final Consolidation Result */}
        <Card withBorder shadow="sm" p="md">
          <Title order={4} mb="md">Gesamtbewertung</Title>
          
          <Grid>
            <Grid.Col span={4}>
              <Stack gap="xs">
                <Text size="sm" fw={500} c="dimmed">Gewichtetes Gesamtergebnis</Text>
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
                <Text size="sm" fw={500} c="dimmed">Abschlusskommentar des Reviewer-Gremiums</Text>
                <Text size="sm" style={{ whiteSpace: 'pre-wrap' }}>
                  {data.final_comment || 'Kein Kommentar verfügbar'}
                </Text>
              </Stack>
            </Grid.Col>
          </Grid>
          
          <Divider my="md" />
          
          <Group gap="xs">
            <Text size="sm" fw={500}>Reviewer:</Text>
            {data.reviewers?.map((reviewer) => (
              <Badge key={reviewer.id} color="blue">
                {reviewer.reviewer_name}
              </Badge>
            ))}
          </Group>
        </Card>

        {/* Discussion Comments Section */}
        <Paper withBorder p="md">
          <Title order={4} mb="md">Notizen zur Besprechung</Title>
          <Stack gap="md">
            <Textarea
              placeholder="Notizen zur Besprechung hinzufügen..."
              value={discussionComment}
              onChange={(e) => setDiscussionComment(e.target.value)}
              minRows={4}
              disabled={saving}
            />
            
            <Button
              onClick={handleSaveComment}
              loading={saving}
              leftSection={<IconDeviceFloppy size={16} />}
            >
              Notizen speichern
            </Button>
          </Stack>
        </Paper>

        {/* Meeting Confirmation Section */}
        <Paper withBorder p="md">
          <Title order={4} mb="md">
            Besprechungsbestätigung
          </Title>
          
          <Alert icon={<IconInfoCircle size={16} />} color="blue" mb="md">
            Bestätigen Sie hier, dass die Besprechung mit dem Mitarbeiter stattgefunden hat. 
            Nach Ihrer Bestätigung kann der Mitarbeiter ebenfalls bestätigen.
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
                        {confirmation.user_type === 'reviewer' ? 'Reviewer' : 'Mitarbeiter'}
                      </Badge>
                      <Text size="sm">
                        {confirmation.user_name || 'Unbekannt'} - {' '}
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

            {!currentUserConfirmed && (
              <Button
                onClick={handleConfirmMeeting}
                loading={confirming}
                leftSection={<IconUserCheck size={16} />}
                color="blue"
              >
                Besprechung bestätigen (als Reviewer)
              </Button>
            )}

            {currentUserConfirmed && (
              <Alert color="green" icon={<IconCheck size={16} />}>
                Sie haben die Besprechung bereits bestätigt.
                {!ownerConfirmed && ' Der Mitarbeiter muss noch bestätigen.'}
              </Alert>
            )}

            {ownerConfirmed && (
              <Alert color="green" icon={<IconCheck size={16} />}>
                Der Mitarbeiter hat die Besprechung bestätigt.
              </Alert>
            )}

            {currentUserConfirmed && ownerConfirmed && (
              <Button
                onClick={handleArchiveAssessment}
                loading={archiving}
                leftSection={<IconArchive size={16} />}
                color="green"
                fullWidth
                mt="md"
              >
                Assessment archivieren
              </Button>
            )}
          </Stack>
        </Paper>
      </Stack>
    </Container>
  );
}
