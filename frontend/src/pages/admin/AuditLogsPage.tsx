import { useState, useEffect } from 'react';
import {
  Container,
  Paper,
  Title,
  Table,
  Badge,
  Group,
  Pagination,
  Text,
  Code,
} from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { adminApi } from '../../services/admin';
import type { AuditLog, ApiError } from '../../types';

export const AuditLogsPage = () => {
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [currentPage, setCurrentPage] = useState(1);

  useEffect(() => {
    loadLogs();
  }, [currentPage]);

  const loadLogs = async () => {
    setIsLoading(true);
    try {
      const data = await adminApi.listAuditLogs(currentPage, 50);
      setLogs(data);
    } catch (error) {
      const apiError = error as ApiError;
      notifications.show({
        title: 'Error',
        message: apiError.error || 'Failed to load audit logs',
        color: 'red',
      });
    } finally {
      setIsLoading(false);
    }
  };

  const getActionColor = (action: string): string => {
    if (action.includes('login')) return 'blue';
    if (action.includes('register')) return 'green';
    if (action.includes('delete') || action.includes('remove')) return 'red';
    if (action.includes('update') || action.includes('assign')) return 'yellow';
    return 'gray';
  };

  return (
    <Container size="xl" my={40}>
      <Paper withBorder shadow="md" p={30} radius="md">
        <Title order={2} mb="xl">Audit Logs</Title>

        <Table striped highlightOnHover>
          <Table.Thead>
            <Table.Tr>
              <Table.Th>Timestamp</Table.Th>
              <Table.Th>User ID</Table.Th>
              <Table.Th>Action</Table.Th>
              <Table.Th>Resource</Table.Th>
              <Table.Th>Details</Table.Th>
              <Table.Th>IP Address</Table.Th>
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {isLoading ? (
              <Table.Tr>
                <Table.Td colSpan={6} style={{ textAlign: 'center' }}>
                  Loading...
                </Table.Td>
              </Table.Tr>
            ) : logs.length === 0 ? (
              <Table.Tr>
                <Table.Td colSpan={6} style={{ textAlign: 'center' }}>
                  No audit logs found
                </Table.Td>
              </Table.Tr>
            ) : (
              logs.map((log) => (
                <Table.Tr key={log.id}>
                  <Table.Td>
                    <Text size="sm">
                      {new Date(log.created_at).toLocaleString('en-US', {
                        year: 'numeric',
                        month: 'short',
                        day: 'numeric',
                        hour: '2-digit',
                        minute: '2-digit',
                        second: '2-digit',
                      })}
                    </Text>
                  </Table.Td>
                  <Table.Td>
                    {log.user_id ? (
                      <Code>{log.user_id}</Code>
                    ) : (
                      <Text c="dimmed" size="sm">N/A</Text>
                    )}
                  </Table.Td>
                  <Table.Td>
                    <Badge color={getActionColor(log.action)} variant="light">
                      {log.action}
                    </Badge>
                  </Table.Td>
                  <Table.Td>
                    <Text size="sm">{log.resource}</Text>
                  </Table.Td>
                  <Table.Td>
                    <Text size="sm" lineClamp={2}>
                      {log.details || '-'}
                    </Text>
                  </Table.Td>
                  <Table.Td>
                    <Code>{log.ip_address || 'N/A'}</Code>
                  </Table.Td>
                </Table.Tr>
              ))
            )}
          </Table.Tbody>
        </Table>

        <Group justify="center" mt="xl">
          <Pagination
            value={currentPage}
            onChange={setCurrentPage}
            total={10} // TODO: Get actual total from backend
          />
        </Group>
      </Paper>
    </Container>
  );
};
