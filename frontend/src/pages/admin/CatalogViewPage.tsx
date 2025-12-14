import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  Container,
  Title,
  Paper,
  Stack,
  Text,
  Loader,
  Alert,
  Badge,
  Group,
  ActionIcon,
  Table,
  Accordion,
} from '@mantine/core';
import { IconAlertCircle, IconArrowLeft } from '@tabler/icons-react';
import { adminApi } from '../../services/admin';
import type { CatalogWithDetails, CatalogPhase } from '../../types';

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

export function CatalogViewPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [catalog, setCatalog] = useState<CatalogWithDetails | null>(null);

  useEffect(() => {
    if (id === 'new') {
      navigate('/admin/catalogs/new/edit', { replace: true });
      return;
    }
    if (id) {
      const catalogId = parseInt(id, 10);
      // Only load if we have a valid numeric ID
      if (!isNaN(catalogId) && catalogId > 0) {
        loadCatalog(catalogId);
      } else {
        setError('Ungültige Katalog-ID');
        setLoading(false);
      }
    }
  }, [id, navigate]);

  const loadCatalog = async (catalogId: number) => {
    try {
      setLoading(true);
      setError(null);
      const data = await adminApi.getCatalog(catalogId);
      setCatalog(data);
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to load catalog');
    } finally {
      setLoading(false);
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
          <Text>Katalog wird geladen...</Text>
        </Stack>
      </Container>
    );
  }

  if (error || !catalog) {
    return (
      <Container size="xl" py="xl">
        <Alert icon={<IconAlertCircle size={16} />} title="Fehler" color="red">
          {error || 'Katalog nicht gefunden'}
        </Alert>
      </Container>
    );
  }

  return (
    <Container size="xl" py="xl">
      <Stack gap="lg">
        <Group justify="space-between">
          <Group>
            <ActionIcon variant="subtle" onClick={() => navigate('/admin/catalogs')}>
              <IconArrowLeft size={20} />
            </ActionIcon>
            <div>
              <Title order={2}>{catalog.name}</Title>
              <Text c="dimmed" size="sm">
                ID: {catalog.id} • Erstellt am:{' '}
                {new Date(catalog.created_at).toLocaleDateString('de-DE')}
              </Text>
            </div>
          </Group>
          <Badge color={getPhaseColor(catalog.phase)}>{getPhaseLabel(catalog.phase)}</Badge>
        </Group>

        <Paper p="md" withBorder>
          <Stack gap="sm">
            <Group>
              <Text fw={500}>Beschreibung:</Text>
              <Text>{catalog.description || 'Keine Beschreibung'}</Text>
            </Group>
            <Group>
              <Text fw={500}>Gültig von:</Text>
              <Text>{formatDate(catalog.valid_from)}</Text>
            </Group>
            <Group>
              <Text fw={500}>Gültig bis:</Text>
              <Text>{formatDate(catalog.valid_until)}</Text>
            </Group>
            <Group>
              <Text fw={500}>Erstellt am:</Text>
              <Text>{formatDate(catalog.created_at)}</Text>
            </Group>
          </Stack>
        </Paper>

        <Title order={3}>Levels</Title>
        <Paper p="md" withBorder>
          {!catalog.levels || catalog.levels.length === 0 ? (
            <Alert color="blue">Keine Levels vorhanden</Alert>
          ) : (
            <Table>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th>Nummer</Table.Th>
                  <Table.Th>Name</Table.Th>
                  <Table.Th>Beschreibung</Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {catalog.levels.map((level) => (
                  <Table.Tr key={level.id}>
                    <Table.Td>{level.level_number}</Table.Td>
                    <Table.Td>{level.name}</Table.Td>
                    <Table.Td>{level.description || '-'}</Table.Td>
                  </Table.Tr>
                ))}
              </Table.Tbody>
            </Table>
          )}
        </Paper>

        <Title order={3}>Kategorien und Pfade</Title>
        {!catalog.categories || catalog.categories.length === 0 ? (
          <Paper p="md" withBorder>
            <Alert color="blue">Keine Kategorien vorhanden</Alert>
          </Paper>
        ) : (
          <Accordion variant="separated">
            {catalog.categories.map((category) => (
              <Accordion.Item key={category.id} value={category.id.toString()}>
                <Accordion.Control>
                  <Group>
                    <Text fw={500}>{category.name}</Text>
                    {category.description && (
                      <Text size="sm" c="dimmed">
                        {category.description}
                      </Text>
                    )}
                  </Group>
                </Accordion.Control>
                <Accordion.Panel>
                  {!category.paths || category.paths.length === 0 ? (
                    <Alert color="blue">Keine Pfade in dieser Kategorie</Alert>
                  ) : (
                    <Stack gap="md">
                      {category.paths.map((path) => (
                        <Paper key={path.id} p="sm" withBorder>
                          <Text fw={500} mb="xs">
                            {path.name}
                          </Text>
                          {path.description && (
                            <Text size="sm" c="dimmed" mb="xs">
                              {path.description}
                            </Text>
                          )}
                          {path.descriptions && path.descriptions.length > 0 && (
                            <Table mt="xs">
                              <Table.Thead>
                                <Table.Tr>
                                  <Table.Th>Level</Table.Th>
                                  <Table.Th>Beschreibung</Table.Th>
                                </Table.Tr>
                              </Table.Thead>
                              <Table.Tbody>
                                {path.descriptions.map((desc) => {
                                  const level = catalog.levels?.find((l) => l.id === desc.level_id);
                                  return (
                                    <Table.Tr key={desc.id}>
                                      <Table.Td>{level?.name || `Level ${desc.level_id}`}</Table.Td>
                                      <Table.Td>{desc.description}</Table.Td>
                                    </Table.Tr>
                                  );
                                })}
                              </Table.Tbody>
                            </Table>
                          )}
                        </Paper>
                      ))}
                    </Stack>
                  )}
                </Accordion.Panel>
              </Accordion.Item>
            ))}
          </Accordion>
        )}
      </Stack>
    </Container>
  );
}
