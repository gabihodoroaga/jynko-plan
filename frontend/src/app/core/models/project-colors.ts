/**
 * Project background colour palette (PLAN-4).
 *
 * Each project can pick one of these light tints in Settings → Project so the
 * project-scoped pages (Overview, Work items, Cycles, Epics, Modules) are easy
 * to tell apart when working across several projects. Keys must stay in sync
 * with `model.ProjectBackgroundColors` in the Go backend.
 */
export type ProjectBackgroundColorKey =
  | 'DEFAULT'
  | 'SKY'
  | 'INDIGO'
  | 'VIOLET'
  | 'ROSE'
  | 'AMBER'
  | 'LIME'
  | 'EMERALD'
  | 'TEAL'
  | 'STONE';

export interface ProjectBackgroundColorOption {
  key: ProjectBackgroundColorKey;
  label: string;
  /** Canvas background applied to project pages. */
  canvas: string;
  /** Slightly stronger swatch / accent colour used in pickers and the sidebar dot. */
  swatch: string;
}

export const PROJECT_BACKGROUND_COLORS: ProjectBackgroundColorOption[] = [
  { key: 'DEFAULT', label: 'Default (Slate)', canvas: '#f8fafc', swatch: '#e2e8f0' },
  { key: 'SKY', label: 'Sky', canvas: '#f0f9ff', swatch: '#7dd3fc' },
  { key: 'INDIGO', label: 'Indigo', canvas: '#eef2ff', swatch: '#a5b4fc' },
  { key: 'VIOLET', label: 'Violet', canvas: '#f5f3ff', swatch: '#c4b5fd' },
  { key: 'ROSE', label: 'Rose', canvas: '#fff1f2', swatch: '#fda4af' },
  { key: 'AMBER', label: 'Amber', canvas: '#fffbeb', swatch: '#fcd34d' },
  { key: 'LIME', label: 'Lime', canvas: '#f7fee7', swatch: '#bef264' },
  { key: 'EMERALD', label: 'Emerald', canvas: '#ecfdf5', swatch: '#6ee7b7' },
  { key: 'TEAL', label: 'Teal', canvas: '#f0fdfa', swatch: '#5eead4' },
  { key: 'STONE', label: 'Stone', canvas: '#f5f5f4', swatch: '#d6d3d1' },
];

export function getProjectBackgroundColor(key?: string | null): ProjectBackgroundColorOption {
  return PROJECT_BACKGROUND_COLORS.find((c) => c.key === key) ?? PROJECT_BACKGROUND_COLORS[0];
}
