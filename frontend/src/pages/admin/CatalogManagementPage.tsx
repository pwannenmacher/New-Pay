import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Container,
  Title,
  Table,
  Badge,
  Button,
  Group,
  Stack,
  Text,
  ActionIcon,
  Loader,
  Alert,
  Menu,
} from '@mantine/core';
import {
  IconPlus,
  IconEdit,
  IconTrash,
  IconEye,
  IconArchive,
  IconChecks,
  IconAlertCircle,
  IconDots,
} from '@tabler/icons-react';
import { adminApi } from '../../services/admin';
import type { CriteriaCatalog, CatalogPhase } from '../../types';

const getPhaseColor = (phase: CatalogPhase): string => {
  switch (phase) {
    case 'draft':
      return 'gray';
    case 'active':
      return 'blue';
    case 'archived':
      return 'green';
    default:
      return 'gray';
  }
};

const getPhaseLabel = (phase: CatalogPhase): string => {
  switch (phase) {
    case 'draft':
      return 'Entwurf';
    case 'active':
      return 'Aktiv';
    case 'archived':
      return 'Archiviert';
    default:
      return phase;
  }
};

export function CatalogManagementPage() {
  const navigate = useNavigate();
  const [catalogs, setCatalogs] = useState<CriteriaCatalog[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadCatalogs = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await adminApi.listCatalogs();
      setCatalogs(data);
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to load catalogs');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadCatalogs();
  }, []);

  const handleDelete = async (catalogId: number, catalogName: string) => {
    if (!confirm(`Kriterienkatalog "${catalogName}" wirklich löschen? Diese Aktion kann nicht rückgängig gemacht werden.`)) {
      return;
    }

    try {
      await adminApi.deleteCatalog(catalogId);
      await loadCatalogs();
    } catch (err: any) {
      alert(err.response?.data?.error || 'Failed to delete catalog');
    }
  };

  const handleTransitionToActive = async (catalogId: number, catalogName: string) => {
    if (!confirm(`Kriterienkatalog "${catalogName}" aktivieren? Der Katalog wird dann für Reviewer und User sichtbar.`)) {
      return;
    }

    try {
      await adminApi.transitionToActive(catalogId);
      await loadCatalogs();
    } catch (err: any) {
      alert(err.response?.data?.error || 'Failed to transition catalog');
    }
  };

  const handleTransitionToArchived = async (catalogId: number, catalogName: string) => {
    if (!confirm(`Kriterienkatalog "${catalogName}" archivieren? Der Katalog kann danach nicht mehr bearbeitet werden.`)) {
      return;
    }

    try {
      await adminApi.transitionToArchived(catalogId);
      await loadCatalogs();
    } catch (err: any) {
      alert(err.response?.data?.error || 'Failed to archive catalog');
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('de-DE');
  };

  if (loading) {
    return (
      <Container size="xl" py="xl">
        <Stack align="center" py="xl">
          <Loader size="lg" />
          <Text>Kriterienkataloge werden geladen...</Text>
        </Stack>
      </Container>
    );
  }

  return (
    <Container size="xl" py="xl">
      <Stack gap="lg">
        <Group justify="space-between">
          <Title order={2}>Kriterienkataloge</Title>
          <Button
            leftSection={<IconPlus size={16} />}
            onClick={() => navigate('/admin/catalogs/new/edit')}
          >
            Neuer Katalog
          </Button>
        </Group>

        {error && (
          <Alert icon={<IconAlertCircle size={16} />} title="Fehler" color="red">
            {error}
          </Alert>
        )}

        {catalogs.length === 0 ? (
          <Alert icon={<IconAlertCircle size={16} />} title="Keine Kataloge" color="blue">
            Es wurden noch keine Kriterienkataloge angelegt.
          </Alert>
        ) : (
          <Table striped highlightOnHover>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Name</Table.Th>
                <Table.Th>Gültig von</Table.Th>
                <Table.Th>Gültig bis</Table.Th>
                <Table.Th>Phase</Table.Th>
                <Table.Th>Erstellt am</Table.Th>
                <Table.Th>Aktionen</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {catalogs.map((catalog) => (
                <Table.Tr key={catalog.id}>
                  <Table.Td>
                    <Text fw={500}>{catalog.name}</Text>
                    {catalog.description && (
                      <Text size="sm" c="dimmed">
                        {catalog.description}
                      </Text>
                    )}
                  </Table.Td>
                  <Table.Td>{formatDate(catalog.valid_from)}</Table.Td>
                  <Table.Td>{formatDate(catalog.valid_until)}</Table.Td>
                  <Table.Td>
                    <Badge color={getPhaseColor(catalog.phase)}>
                      {getPhaseLabel(catalog.phase)}
                    </Badge>
                  </Table.Td>
                  <Table.Td>{formatDate(catalog.created_at)}</Table.Td>
                  <Table.Td>
                    <Group gap="xs">
                      <ActionIcon
                        variant="light"
                        color="blue"
                        onClick={() => navigate(`/admin/catalogs/${catalog.id}`)}
                      >
                        <IconEye size={16} />
                      </ActionIcon>
                      
                      {catalog.phase !== 'archived' && (
                        <ActionIcon
                          variant="light"
                          color="cyan"
                          onClick={() => navigate(`/admin/catalogs/${catalog.id}/edit`)}
                        >
                          <IconEdit size={16} />
                        </ActionIcon>
                      )}

                      <Menu shadow="md" width={200}>
                        <Menu.Target>
                          <ActionIcon variant="light">
                            <IconDots size={16} />
                          </ActionIcon>
                        </Menu.Target>

                        <Menu.Dropdown>
                          {catalog.phase === 'draft' && (
                            <>
                              <Menu.Item
                                leftSection={<IconChecks size={16} />}
                                onClick={() => handleTransitionToActive(catalog.id, catalog.name)}
                              >
                                Aktivieren
                              </Menu.Item>
                              <Menu.Item
                                leftSection={<IconTrash size={16} />}
                                color="red"
                                onClick={() => handleDelete(catalog.id, catalog.name)}
                              >
                                Löschen
                              </Menu.Item>
                            </>
                          )}

                          {catalog.phase === 'active' && (
                            <Menu.Item
                              leftSection={<IconArchive size={16} />}
                              onClick={() => handleTransitionToArchived(catalog.id, catalog.name)}
                            >
                              Archivieren
                            </Menu.Item>
                          )}

                          {catalog.phase === 'archived' && (
                            <Menu.Item disabled>
                              Archiviert (keine Aktionen)
                            </Menu.Item>
                          )}
                        </Menu.Dropdown>
                      </Menu>
                    </Group>
                  </Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        )}
      </Stack>
    </Container>
  );
}
