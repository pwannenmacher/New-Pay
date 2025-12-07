import { useState, useEffect } from 'react';
import {
  Container,
  Paper,
  Title,
  Table,
  Badge,
  Button,
  Group,
  Pagination,
  Select,
  Modal,
  Stack,
  Text,
  ActionIcon,
  Tooltip,
} from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { IconUserPlus, IconUserMinus } from '@tabler/icons-react';
import { adminApi } from '../../services/admin';
import type { UserWithRoles, Role, ApiError } from '../../types';

export const UserManagementPage = () => {
  const [users, setUsers] = useState<UserWithRoles[]>([]);
  const [roles, setRoles] = useState<Role[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [currentPage, setCurrentPage] = useState(1);
  const [modalOpened, setModalOpened] = useState(false);
  const [selectedUser, setSelectedUser] = useState<UserWithRoles | null>(null);
  const [selectedRoleId, setSelectedRoleId] = useState<string | null>(null);
  const [actionType, setActionType] = useState<'assign' | 'remove'>('assign');
  const [actionLoading, setActionLoading] = useState(false);

  useEffect(() => {
    loadUsers();
    loadRoles();
  }, [currentPage]);

  const loadUsers = async () => {
    setIsLoading(true);
    try {
      const data = await adminApi.listUsers(currentPage, 20);
      setUsers(data);
    } catch (error) {
      const apiError = error as ApiError;
      notifications.show({
        title: 'Error',
        message: apiError.error || 'Failed to load users',
        color: 'red',
      });
    } finally {
      setIsLoading(false);
    }
  };

  const loadRoles = async () => {
    try {
      const data = await adminApi.listRoles();
      setRoles(data);
    } catch (error) {
      console.error('Failed to load roles:', error);
    }
  };

  const openAssignRoleModal = (user: UserWithRoles) => {
    setSelectedUser(user);
    setActionType('assign');
    setSelectedRoleId(null);
    setModalOpened(true);
  };

  const openRemoveRoleModal = (user: UserWithRoles) => {
    setSelectedUser(user);
    setActionType('remove');
    setSelectedRoleId(null);
    setModalOpened(true);
  };

  const handleRoleAction = async () => {
    if (!selectedUser || !selectedRoleId) {
      notifications.show({
        title: 'Error',
        message: 'Please select a role',
        color: 'red',
      });
      return;
    }

    setActionLoading(true);
    try {
      if (actionType === 'assign') {
        await adminApi.assignRole({
          user_id: selectedUser.id,
          role_id: parseInt(selectedRoleId),
        });
        notifications.show({
          title: 'Success',
          message: 'Role assigned successfully',
          color: 'green',
        });
      } else {
        await adminApi.removeRole({
          user_id: selectedUser.id,
          role_id: parseInt(selectedRoleId),
        });
        notifications.show({
          title: 'Success',
          message: 'Role removed successfully',
          color: 'green',
        });
      }
      setModalOpened(false);
      loadUsers();
    } catch (error) {
      const apiError = error as ApiError;
      notifications.show({
        title: 'Error',
        message: apiError.error || `Failed to ${actionType} role`,
        color: 'red',
      });
    } finally {
      setActionLoading(false);
    }
  };

  const availableRoles = actionType === 'assign'
    ? roles.filter(role => !selectedUser?.roles?.some(ur => ur.id === role.id))
    : selectedUser?.roles || [];

  return (
    <Container size="xl" my={40}>
      <Paper withBorder shadow="md" p={30} radius="md">
        <Title order={2} mb="xl">User Management</Title>

        <Table striped highlightOnHover>
          <Table.Thead>
            <Table.Tr>
              <Table.Th>ID</Table.Th>
              <Table.Th>Name</Table.Th>
              <Table.Th>Email</Table.Th>
              <Table.Th>Roles</Table.Th>
              <Table.Th>Status</Table.Th>
              <Table.Th>Actions</Table.Th>
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {isLoading ? (
              <Table.Tr>
                <Table.Td colSpan={6} style={{ textAlign: 'center' }}>
                  Loading...
                </Table.Td>
              </Table.Tr>
            ) : users.length === 0 ? (
              <Table.Tr>
                <Table.Td colSpan={6} style={{ textAlign: 'center' }}>
                  No users found
                </Table.Td>
              </Table.Tr>
            ) : (
              users.map((user) => (
                <Table.Tr key={user.id}>
                  <Table.Td>{user.id}</Table.Td>
                  <Table.Td>{user.first_name} {user.last_name}</Table.Td>
                  <Table.Td>{user.email}</Table.Td>
                  <Table.Td>
                    <Group gap="xs">
                      {user.roles && user.roles.length > 0 ? (
                        user.roles.map((role) => (
                          <Badge key={role.id} variant="light">
                            {role.name}
                          </Badge>
                        ))
                      ) : (
                        <Text c="dimmed" size="sm">No roles</Text>
                      )}
                    </Group>
                  </Table.Td>
                  <Table.Td>
                    <Badge color={user.is_active ? 'green' : 'red'} variant="light">
                      {user.is_active ? 'Active' : 'Inactive'}
                    </Badge>
                  </Table.Td>
                  <Table.Td>
                    <Group gap="xs">
                      <Tooltip label="Assign Role">
                        <ActionIcon
                          color="blue"
                          variant="light"
                          onClick={() => openAssignRoleModal(user)}
                        >
                          <IconUserPlus size={16} />
                        </ActionIcon>
                      </Tooltip>
                      <Tooltip label="Remove Role">
                        <ActionIcon
                          color="red"
                          variant="light"
                          onClick={() => openRemoveRoleModal(user)}
                          disabled={!user.roles || user.roles.length === 0}
                        >
                          <IconUserMinus size={16} />
                        </ActionIcon>
                      </Tooltip>
                    </Group>
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
            total={10}
          />
        </Group>
      </Paper>

      <Modal
        opened={modalOpened}
        onClose={() => setModalOpened(false)}
        title={actionType === 'assign' ? 'Assign Role' : 'Remove Role'}
        centered
      >
        <Stack>
          <Text size="sm">
            {actionType === 'assign' ? 'Assign a role to' : 'Remove a role from'}{' '}
            <strong>{selectedUser?.first_name} {selectedUser?.last_name}</strong>
          </Text>
          
          <Select
            label="Role"
            placeholder="Select a role"
            value={selectedRoleId}
            onChange={setSelectedRoleId}
            data={availableRoles.map((role) => ({
              value: role.id.toString(),
              label: role.name,
            }))}
            required
          />

          <Group justify="flex-end">
            <Button variant="subtle" onClick={() => setModalOpened(false)}>
              Cancel
            </Button>
            <Button
              onClick={handleRoleAction}
              loading={actionLoading}
              color={actionType === 'assign' ? 'blue' : 'red'}
            >
              {actionType === 'assign' ? 'Assign' : 'Remove'}
            </Button>
          </Group>
        </Stack>
      </Modal>
    </Container>
  );
};
