import { h, type Component } from 'vue'
import { NIcon } from 'naive-ui'
import {
  BuildOutline,
  DocumentTextOutline,
  GridOutline,
  MenuOutline,
  PeopleOutline,
  PersonOutline,
  SettingsOutline,
  SpeedometerOutline,
} from '@vicons/ionicons5'

const icons: Record<string, Component> = {
  Avatar: PersonOutline,
  Document: DocumentTextOutline,
  Menu: MenuOutline,
  Monitor: GridOutline,
  Odometer: SpeedometerOutline,
  Setting: SettingsOutline,
  Tools: BuildOutline,
  User: PersonOutline,
  UserFilled: PeopleOutline,
}

export function resolveIcon(name?: string): Component {
  return icons[name || ''] || GridOutline
}

export function renderIcon(icon: Component) {
  return () => h(NIcon, null, { default: () => h(icon) })
}
