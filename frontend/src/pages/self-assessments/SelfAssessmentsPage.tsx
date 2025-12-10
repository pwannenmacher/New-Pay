import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Container,
  Title,
  Text,
  Paper,
  Group,
  Button,
  Stack,
  Badge,
  Table,
  Modal,
  Select,
  Alert,
  LoadingOverlay,
} from '@mantine/core';
import {
  IconPlus,
  IconFileCheck,
  IconClock,
  IconCheck,
  IconMessageCircle,
  IconArchive,
  IconX,
  IconAlertCircle,
} from '@tabler/icons-react';
import { selfAssessmentService } from '../../services/selfAssessment';
import type { SelfAssessment, CriteriaCatalog } from '../../types';
import { notifications } from '@mantine/notifications';

const statusConfig = {
  draft: { label: 'Entwurf', color: 'gray', icon: IconClock },
  submitted: { label: 'Eingereicht', color: 'blue', icon: IconFileCheck },
  in_review: { label: 'In Prüfung', color: 'yellow', icon: IconClock },
  reviewed: { label: 'Geprüft', color: 'orange', icon: IconCheck },
  discussion: { label: 'Besprechung', color: 'violet', icon: IconMessageCircle },
  archived: { label: 'Archiviert', color: 'green', icon: IconArchive },
  closed: { label: 'Geschlossen', color: 'red', icon: IconX },
};

export default function SelfAssessmentsPage() {
  const navigate = useNavigate();
  const [assessments, setAssessments] = useState<SelfAssessment[]>([]);
  const [activeCatalogs, setActiveCatalogs] = useState<CriteriaCatalog[]>([]);
  const [loading, setLoading] = useState(true);
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [selectedCatalogId, setSelectedCatalogId] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    try {
      setLoading(true);
      const [assessmentsData, catalogsData] = await Promise.all([
        selfAssessmentService.getMySelfAssessments(),
        selfAssessmentService.getActiveCatalogs(),
      ]);
      setAssessments(Array.isArray(assessmentsData) ? assessmentsData : []);
      setActiveCatalogs(Array.isArray(catalogsData) ? catalogsData : []);
    } catch (error) {
      console.error('Error loading data:', error);
      setAssessments([]);
      setActiveCatalogs([]);
      notifications.show({
        title: 'Fehler',
        message: 'Daten konnten nicht geladen werden',
        color: 'red',
      });
    } finally {
      setLoading(false);
    }
  };

  const handleCreateAssessment = async () => {
    if (!selectedCatalogId) return;

    try {
      setCreating(true);
      const newAssessment = await selfAssessmentService.createSelfAssessment(
        parseInt(selectedCatalogId)
      );
      notifications.show({
        title: 'Erfolg',
        message: 'Selbsteinschätzung wurde erstellt',
        color: 'green',
      });
      setCreateModalOpen(false);
      setSelectedCatalogId(null);
      navigate(`/self-assessments/${newAssessment.id}`);
    } catch (error: any) {
      console.error('Error creating assessment:', error);
      notifications.show({
        title: 'Fehler',
        message: error.response?.data?.error || 'Selbsteinschätzung konnte nicht erstellt werden',
        color: 'red',
      });
    } finally {
      setCreating(false);
    }
  };

  const formatDate = (dateString?: string) => {
    if (!dateString) return '-';
    return new Date(dateString).toLocaleDateString('de-DE', {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
    });
  };

  const getStatusBadge = (status: string) => {
    const config = statusConfig[status as keyof typeof statusConfig] || {
      label: status,
      color: 'gray',
      icon: IconClock,
    };
    const Icon = config.icon;
    return (
      <Badge color={config.color} leftSection={<Icon size={14} />}>
        {config.label}
      </Badge>
    );
  };

  const canCreateForCatalog = (catalogId: number) => {
    return !assessments.some((a) => a.catalog_id === catalogId);
  };

  const availableCatalogs = activeCatalogs.filter((c) => canCreateForCatalog(c.id));

  return (
    <Container size="xl" py="xl">
      <LoadingOverlay visible={loading} />

      <Stack gap="lg">
        <Group justify="space-between">
          <div>
            <Title order={1}>Meine Selbsteinschätzungen</Title>
            <Text c="dimmed" size="sm">
              Verwalten Sie Ihre Selbsteinschätzungen zu aktiven Kriterienkatalogen
            </Text>
          </div>
          <Button
            leftSection={<IconPlus size={16} />}
            onClick={() => setCreateModalOpen(true)}
            disabled={availableCatalogs.length === 0}
          >
            Neue Selbsteinschätzung
          </Button>
        </Group>

        {availableCatalogs.length === 0 && assessments.length === 0 && (
          <Alert icon={<IconAlertCircle size={16} />} title="Keine Kataloge verfügbar" color="blue">
            Zurzeit sind keine aktiven Kriterienkataloge verfügbar, für die Sie eine
            Selbsteinschätzung erstellen können.
          </Alert>
        )}

        {assessments.length > 0 ? (
          <Paper shadow="sm" p="md" withBorder>
            <Table striped highlightOnHover>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th>Katalog-ID</Table.Th>
                  <Table.Th>Status</Table.Th>
                  <Table.Th>Erstellt am</Table.Th>
                  <Table.Th>Eingereicht am</Table.Th>
                  <Table.Th>Aktualisiert am</Table.Th>
                  <Table.Th>Aktionen</Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {assessments.map((assessment) => (
                  <Table.Tr key={assessment.id}>
                    <Table.Td>{assessment.catalog_id}</Table.Td>
                    <Table.Td>{getStatusBadge(assessment.status)}</Table.Td>
                    <Table.Td>{formatDate(assessment.created_at)}</Table.Td>
                    <Table.Td>{formatDate(assessment.submitted_at)}</Table.Td>
                    <Table.Td>{formatDate(assessment.updated_at)}</Table.Td>
                    <Table.Td>
                      <Button
                        size="xs"
                        variant="light"
                        onClick={() => navigate(`/self-assessments/${assessment.id}`)}
                      >
                        Details
                      </Button>
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
                <IconFileCheck size={48} stroke={1.5} />
                <Text c="dimmed">Sie haben noch keine Selbsteinschätzungen erstellt</Text>
                {availableCatalogs.length > 0 && (
                  <Button onClick={() => setCreateModalOpen(true)} leftSection={<IconPlus size={16} />}>
                    Erste Selbsteinschätzung erstellen
                  </Button>
                )}
              </Stack>
            </Paper>
          )
        )}
      </Stack>

      <Modal
        opened={createModalOpen}
        onClose={() => {
          setCreateModalOpen(false);
          setSelectedCatalogId(null);
        }}
        title="Neue Selbsteinschätzung erstellen"
      >
        <Stack>
          <Select
            label="Kriterienkatalog"
            placeholder="Wählen Sie einen Katalog"
            data={availableCatalogs.map((catalog) => ({
              value: catalog.id.toString(),
              label: `${catalog.name} (${formatDate(catalog.valid_from)} - ${formatDate(catalog.valid_until)})`,
            }))}
            value={selectedCatalogId}
            onChange={setSelectedCatalogId}
          />

          <Group justify="flex-end" mt="md">
            <Button variant="subtle" onClick={() => setCreateModalOpen(false)} disabled={creating}>
              Abbrechen
            </Button>
            <Button
              onClick={handleCreateAssessment}
              disabled={!selectedCatalogId}
              loading={creating}
            >
              Erstellen
            </Button>
          </Group>
        </Stack>
      </Modal>
    </Container>
  );
}
