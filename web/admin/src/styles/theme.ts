import type { GlobalThemeOverrides } from 'naive-ui'

export const themeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#0f766e',
    primaryColorHover: '#0d9488',
    primaryColorPressed: '#115e59',
    primaryColorSuppl: '#14b8a6',
    infoColor: '#2563eb',
    successColor: '#15803d',
    warningColor: '#b45309',
    errorColor: '#b91c1c',
    borderRadius: '6px',
    borderRadiusSmall: '4px',
    fontFamily:
      'Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
    fontSize: '14px',
    textColorBase: '#17202a',
    bodyColor: '#f4f6f8',
  },
  Button: {
    borderRadiusMedium: '6px',
    borderRadiusSmall: '5px',
    fontWeight: '600',
  },
  Card: {
    borderRadius: '8px',
    paddingMedium: '20px',
    titleFontSizeMedium: '16px',
  },
  DataTable: {
    borderRadius: '6px',
    thColor: '#f7f8fa',
    thColorHover: '#f7f8fa',
    tdColorHover: '#f8fafb',
    thFontWeight: '600',
  },
  Dialog: {
    borderRadius: '8px',
  },
  Input: {
    borderRadius: '6px',
  },
  Select: {
    peers: {
      InternalSelection: {
        borderRadius: '6px',
      },
    },
  },
  Tabs: {
    tabBorderRadius: '6px',
  },
}
