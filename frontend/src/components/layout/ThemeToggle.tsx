import { ActionIcon, Menu, Group, Text } from '@mantine/core';
import { IconSun, IconMoon, IconDeviceDesktop } from '@tabler/icons-react';
import { useTheme } from '../../contexts/ThemeContext';
import { useMantineColorScheme } from '@mantine/core';

export function ThemeToggle() {
  const { themeMode, toggleTheme } = useTheme();
  const { colorScheme } = useMantineColorScheme();

  const getIcon = () => {
    if (themeMode === 'auto') {
      return <IconDeviceDesktop size={20} />;
    }
    return colorScheme === 'dark' ? <IconMoon size={20} /> : <IconSun size={20} />;
  };

  return (
    <Menu shadow="md" width={200}>
      <Menu.Target>
        <ActionIcon variant="default" size="lg">
          {getIcon()}
        </ActionIcon>
      </Menu.Target>

      <Menu.Dropdown>
        <Menu.Label>Theme-Modus</Menu.Label>
        <Menu.Item
          leftSection={<IconSun size={16} />}
          onClick={() => toggleTheme('light')}
          style={{
            backgroundColor: themeMode === 'light' ? 'var(--mantine-color-blue-light)' : undefined,
          }}
        >
          <Group justify="space-between">
            <Text>Hell</Text>
            {themeMode === 'light' && (
              <Text size="xs" c="dimmed">
                ✓
              </Text>
            )}
          </Group>
        </Menu.Item>
        <Menu.Item
          leftSection={<IconMoon size={16} />}
          onClick={() => toggleTheme('dark')}
          style={{
            backgroundColor: themeMode === 'dark' ? 'var(--mantine-color-blue-light)' : undefined,
          }}
        >
          <Group justify="space-between">
            <Text>Dunkel</Text>
            {themeMode === 'dark' && (
              <Text size="xs" c="dimmed">
                ✓
              </Text>
            )}
          </Group>
        </Menu.Item>
        <Menu.Item
          leftSection={<IconDeviceDesktop size={16} />}
          onClick={() => toggleTheme('auto')}
          style={{
            backgroundColor: themeMode === 'auto' ? 'var(--mantine-color-blue-light)' : undefined,
          }}
        >
          <Group justify="space-between">
            <Text>System</Text>
            {themeMode === 'auto' && (
              <Text size="xs" c="dimmed">
                ✓
              </Text>
            )}
          </Group>
        </Menu.Item>
      </Menu.Dropdown>
    </Menu>
  );
}
