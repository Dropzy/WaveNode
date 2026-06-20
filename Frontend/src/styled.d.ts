import 'styled-components'
import type { AppTheme } from './styles/themes'

declare module 'styled-components' {
  export interface DefaultTheme {
    name: AppTheme['name']
    label: AppTheme['label']
    description: AppTheme['description']
    colors: AppTheme['colors']
  }
}
