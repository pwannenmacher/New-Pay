import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Container,
  Title,
  Text,
  Paper,
  Group,
  Stack,
  Badge,
  Table,
  TextInput,
  Select,
  Button,
  LoadingOverlay,
} from '@mantine/core';
import {
  IconSearch,
  IconFilter,
  IconArchive,
  IconEye,
} from '@tabler/icons-react';
import { DateInput } from '@mantine/dates';
import { selfAssessmentService } from '../../services/selfAssessment';
import type { SelfAssessment, Role } from '../../types';
import { notifications } from '@mantine/notifications';
import { useAuth } from '../../contexts/AuthContext';

// Helper function to parse German date format DD.MM.YYYY
const dateParser = (input: string): Date | null => {
  if (!input || input.trim() === '') return null;
  const parts = input.trim().split('.');
  if (parts.length === 3) {
    const day = parseInt(parts[0], 10);
    const month = parseInt(parts[1], 10) - 1;
    const year = parseInt(parts[2], 10);
    if (!isNaN(day) && !isNaN(month) && !isNaN(year)) {
      const date = new Date(year, month, day);
      if (date.getDate() === day && date.getMonth() === month && date.getFullYear() === year) {
        return date;
      }
    }
  }
  return null;
};

export function ReviewCompletedAssessmentsPage() {
  const navigate = useNavigate();
  const { user } = useAuth();
  const isAdmin = user?.roles?.some((role: Role) => role.name === 'admin');
  const [assessments, setAssessments] = useState<SelfAssessment[]>([]);
  const [allAssessments, setAllAssessments] = useState<SelfAssessment[]>([]);
  const [loading, setLoading] = useState(true);
  const [username, setUsername] = useState('');
  const [catalogId, setCatalogId] = useState<string>('');
  const [fromDate, setFromDate] = useState<string | null>(null);
  const [toDate, setToDate] = useState<string | null>(null);
  const [fromSubmittedDate, setFromSubmittedDate] = useState<string | null>(null);
  const [toSubmittedDate, setToSubmittedDate] = useState<string | null>(null);

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    try {
      setLoading(true);
      const filters: any = {};
      if (username) filters.username = username;
      if (catalogId) filters.catalog_id = parseInt(catalogId);
      if (fromDate) filters.from_date = fromDate;
      if (toDate) filters.to_date = toDate;
      if (fromSubmittedDate) filters.from_submitted_date = fromSubmittedDate;
      if (toSubmittedDate) filters.to_submitted_date = toSubmittedDate;

      const data = await selfAssessmentService.getCompletedAssessmentsForReview(filters);
      const assessmentsList = Array.isArray(data) ? data : [];
      setAssessments(assessmentsList);
      
      // Store all assessments for catalog dropdown
      if (!allAssessments.length) {
        setAllAssessments(assessmentsList);
      }
    } catch (error: any) {
      console.error('Error loading assessments:', error);
      notifications.show({
        title: 'Fehler',
        message: error.response?.data?.error || 'Fehler beim Laden der Selbsteinschätzungen',
        color: 'red',
      });
    } finally {
      setLoading(false);
    }
  };

  const handleFilter = () => {
    loadData();
  };

  const handleReset = () => {
    setUsername('');
    setCatalogId('');
    setFromDate(null);
    setToDate(null);
    setFromSubmittedDate(null);
    setToSubmittedDate(null);
    // Reload with no filters
    setTimeout(() => loadData(), 100);
  };

  const formatDate = (dateString?: string) => {
    if (!dateString) return '-';
    return new Date(dateString).toLocaleString('de-DE', {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  // Get unique catalogs from all assessments
  const catalogOptions = Array.from(
    new Map(
      allAssessments
        .filter((a) => a.catalog_id && a.catalog_name)
        .map((a) => [a.catalog_id, { value: String(a.catalog_id), label: a.catalog_name || `Katalog ${a.catalog_id}` }])
    ).values()
  );

  return (
    <Container size="xl" py="xl">
      <LoadingOverlay visible={loading} />

      <Stack gap="lg">
        <div>
          <Title order={1}>Abgeschlossene Selbsteinschätzungen</Title>
          <Text c="dimmed" size="sm">
            {isAdmin 
              ? 'Übersicht aller archivierten Selbsteinschätzungen'
              : 'Übersicht Ihrer archivierten Selbsteinschätzungen als Reviewer'}
          </Text>
        </div>

        <Paper shadow="sm" p="md" withBorder>
          <Stack gap="md">
            <Group align="flex-end" gap="md">
              <TextInput
                label="Benutzer"
                placeholder="E-Mail, Vorname oder Nachname"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                leftSection={<IconSearch size={16} />}
                style={{ flex: 1 }}
              />
              <Select
                label="Katalog"
                placeholder="Alle Kataloge"
                value={catalogId}
                onChange={(value) => setCatalogId(value || '')}
                data={[
                  { value: '', label: 'Alle' },
                  ...catalogOptions,
                ]}
                clearable
                searchable
                style={{ minWidth: 200 }}
              />
            </Group>
            <Group align="flex-end" gap="md">
              <DateInput
                label="Erstellt von"
                placeholder="Datum wählen (DD.MM.YYYY)"
                value={fromDate}
                onChange={setFromDate}
                dateParser={dateParser}
                valueFormat="DD.MM.YYYY"
                clearable
                readOnly={false}
                style={{ minWidth: 200 }}
              />
              <DateInput
                label="Erstellt bis"
                placeholder="Datum wählen (DD.MM.YYYY)"
                value={toDate}
                onChange={setToDate}
                dateParser={dateParser}
                valueFormat="DD.MM.YYYY"
                clearable
                readOnly={false}
                style={{ minWidth: 200 }}
              />
              <DateInput
                label="Eingereicht von"
                placeholder="Datum wählen (DD.MM.YYYY)"
                value={fromSubmittedDate}
                onChange={setFromSubmittedDate}
                dateParser={dateParser}
                valueFormat="DD.MM.YYYY"
                clearable
                readOnly={false}
                style={{ minWidth: 200 }}
              />
              <DateInput
                label="Eingereicht bis"
                placeholder="Datum wählen (DD.MM.YYYY)"
                value={toSubmittedDate}
                onChange={setToSubmittedDate}
                dateParser={dateParser}
                valueFormat="DD.MM.YYYY"
                clearable
                readOnly={false}
                style={{ minWidth: 200 }}
              />
            </Group>
            <Group>
              <Button leftSection={<IconFilter size={16} />} onClick={handleFilter}>
                Filtern
              </Button>
              <Button variant="subtle" onClick={handleReset}>
                Zurücksetzen
              </Button>
            </Group>
          </Stack>
        </Paper>

        {assessments.length > 0 ? (
          <Paper shadow="sm" p="md" withBorder>
            <Table striped highlightOnHover>
              <Table.Thead>
                <Table.Tr>
                  {isAdmin && <Table.Th>ID</Table.Th>}
                  <Table.Th>Katalog</Table.Th>
                  <Table.Th>Benutzer</Table.Th>
                  <Table.Th>Status</Table.Th>
                  <Table.Th>Reviews</Table.Th>
                  <Table.Th>Erstellt am</Table.Th>
                  <Table.Th>Eingereicht am</Table.Th>
                  <Table.Th>Archiviert am</Table.Th>
                  <Table.Th>Aktionen</Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {assessments.map((assessment) => (
                  <Table.Tr key={assessment.id}>
                    {isAdmin && <Table.Td>{assessment.id}</Table.Td>}
                    <Table.Td>
                      <div>
                        <Text size="sm">{assessment.catalog_name || 'Unbekannt'}</Text>
                        {isAdmin && (
                          <Text size="xs" c="dimmed">
                            ID: {assessment.catalog_id}
                          </Text>
                        )}
                      </div>
                    </Table.Td>
                    <Table.Td>
                      <div>
                        <Text size="sm">{assessment.user_name || 'Unbekannt'}</Text>
                        <Text size="xs" c="dimmed">
                          {assessment.user_email || 'keine E-Mail'}
                        </Text>
                      </div>
                    </Table.Td>
                    <Table.Td>
                      <Badge color="gray" leftSection={<IconArchive size={14} />}>
                        Archiviert
                      </Badge>
                    </Table.Td>
                    <Table.Td>
                      <Group gap="xs">
                        <Badge size="sm" variant="light" color="blue">
                          {assessment.reviews_started || 0} begonnen
                        </Badge>
                        <Badge size="sm" variant="light" color="green">
                          {assessment.reviews_completed || 0} abgeschlossen
                        </Badge>
                      </Group>
                    </Table.Td>
                    <Table.Td>{formatDate(assessment.created_at)}</Table.Td>
                    <Table.Td>{formatDate(assessment.submitted_at)}</Table.Td>
                    <Table.Td>{formatDate(assessment.archived_at)}</Table.Td>
                    <Table.Td>
                      <Group gap="xs">
                        <Button
                          size="xs"
                          variant="light"
                          leftSection={<IconEye size={14} />}
                          onClick={() => navigate(`/review/discussion/${assessment.id}`)}
                        >
                          Ansehen
                        </Button>
                        <Button
                          size="xs"
                          variant="subtle"
                          onClick={() => navigate(`/review/consolidation/${assessment.id}`)}
                        >
                          Details
                        </Button>
                      </Group>
                    </Table.Td>
                  </Table.Tr>
                ))}
              </Table.Tbody>
            </Table>
          </Paper>
        ) : (
          !loading && (
            <Paper shadow="sm" p="xl" withBorder>
              <Stack align="center" gap="md">
                <Text c="dimmed">Keine abgeschlossenen Selbsteinschätzungen gefunden</Text>
              </Stack>
            </Paper>
          )
        )}
      </Stack>
    </Container>
  );
}
