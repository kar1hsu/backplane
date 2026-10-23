import type { Component } from 'vue'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'

const icons = ElementPlusIconsVue as Record<string, Component>

// resolveIcon maps a menu's icon name to an Element Plus icon component.
// Names come from the database, so anything that is not a known icon resolves
// to nothing — <component :is> would otherwise render any registered component
// or HTML tag, and an invalid tag name breaks the whole sidebar.
export function resolveIcon(name?: string): Component | undefined {
  return name && Object.prototype.hasOwnProperty.call(icons, name) ? icons[name] : undefined
}
