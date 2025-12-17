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
} from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import discussionService, { type DiscussionResult } from '../../services/discussion';

export function ReviewDiscussionPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const assessmentId = parseInt(id || '0');

  const [loading, setLoading] = useState(true);
  const [data, setData] = useState<DiscussionResult | null>(null);
  const [discussionComment, setDiscussionComment] = useState('');
  const [approved, setApproved] = useState(false);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    loadData();
  }, [assessmentId]);

  const loadData = async () => {
    try {
      setLoading(true);
      const discussionData = await discussionService.getDiscussionResult(assessmentId);
      setData(discussionData);
      setDiscussionComment(discussionData.discussion_note || '');
      setApproved(!!discussionData.user_approved_at);
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
      await discussionService.updateDiscussionNote(assessmentId, discussionComment, approved);
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
            
            <Checkbox
              label="Ich habe die Bewertung mit dem Mitarbeiter besprochen und bestätige das Ergebnis"
              checked={approved}
              onChange={(e) => setApproved(e.currentTarget.checked)}
              disabled={saving}
            />
            
            <Group>
              <Button
                onClick={handleSaveComment}
                loading={saving}
                leftSection={<IconDeviceFloppy size={16} />}
              >
                Speichern
              </Button>
              
              {data.user_approved_at && (
                <Badge color="green" leftSection={<IconCheck size={12} />}>
                  Bestätigt am {new Date(data.user_approved_at).toLocaleString('de-DE')}
                </Badge>
              )}
            </Group>
          </Stack>
        </Paper>
      </Stack>
    </Container>
  );
}
