export type ProjectRole = 'ADMIN' | 'MEMBER' | 'VIEWER';

export type WorkItemStatus =
  | 'BACKLOG'
  | 'TODO'
  | 'IN_PROGRESS'
  | 'IN_REVIEW'
  | 'DONE'
  | 'CANCELLED';

export type WorkItemPriority = 'NONE' | 'LOW' | 'MEDIUM' | 'HIGH' | 'URGENT';

export type WorkItemViewMode =
  | 'LIST'
  | 'BOARD'
  | 'CALENDAR'
  | 'TIMELINE'
  | 'SPREADSHEET';

export type EpicStatus = 'PLANNED' | 'IN_PROGRESS' | 'COMPLETED' | 'CANCELLED';

export interface UserContext {
  user_id: string;
  email: string;
  name: string;
  avatar_url?: string;
  avatar_color?: string;
}

export interface Project {
  id: string;
  name: string;
  key_prefix: string;
  description: string;
  item_seq: number;
  member_ids: string[];
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface ProjectMember {
  user_id: string;
  email: string;
  name: string;
  role: ProjectRole;
  joined_at: string;
}

export interface Module {
  id: string;
  project_id: string;
  name: string;
  description: string;
  lead_user_id: string;
  created_at: string;
  updated_at: string;
}

export interface Epic {
  id: string;
  project_id: string;
  name: string;
  description: string;
  status: EpicStatus;
  start_date?: string;
  target_date?: string;
  created_at: string;
  updated_at: string;
}

export interface Cycle {
  id: string;
  project_id: string;
  name: string;
  description: string;
  start_date: string;
  end_date: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface Label {
  id: string;
  project_id: string;
  name: string;
  color: string;
  created_at: string;
}

export interface WorkItem {
  id: string;
  project_id: string;
  key: string;
  sequence_id: number;
  module_id?: string;
  epic_id?: string;
  cycle_id?: string;
  title: string;
  description: string;
  status: WorkItemStatus;
  priority: WorkItemPriority;
  assignee_id?: string;
  reporter_id: string;
  label_ids: string[];
  estimate?: number;
  start_date?: string;
  due_date?: string;
  sub_item_count: number;
  completed_sub_count: number;
  created_at: string;
  updated_at: string;
}

export interface SubWorkItem {
  id: string;
  parent_item_id: string;
  title: string;
  is_completed: boolean;
  assignee_id?: string;
  created_at: string;
  updated_at: string;
}

export interface Comment {
  id: string;
  work_item_id: string;
  author_id: string;
  author_name: string;
  body: string;
  created_at: string;
  updated_at: string;
}

export interface WorkItemFilterState {
  status: string;
  priority: string;
  module_id: string;
  epic_id: string;
  cycle_id: string;
  assignee_id: string;
  label_id: string;
  search: string;
}

export const ALL_STATUSES: { value: WorkItemStatus; label: string }[] = [
  { value: 'BACKLOG', label: 'Backlog' },
  { value: 'TODO', label: 'Todo' },
  { value: 'IN_PROGRESS', label: 'In Progress' },
  { value: 'IN_REVIEW', label: 'In Review' },
  { value: 'DONE', label: 'Done' },
  { value: 'CANCELLED', label: 'Cancelled' },
];

export const ALL_PRIORITIES: { value: WorkItemPriority; label: string }[] = [
  { value: 'URGENT', label: 'Urgent' },
  { value: 'HIGH', label: 'High' },
  { value: 'MEDIUM', label: 'Medium' },
  { value: 'LOW', label: 'Low' },
  { value: 'NONE', label: 'None' },
];

export type StickyColor = 'BANANA' | 'FLAMINGO' | 'MINT' | 'LAGOON' | 'LAVENDER' | 'PEACH';

export interface Sticky {
  id: string;
  project_id: string;
  content: string;
  color: StickyColor;
  is_pinned: boolean;
  author_id: string;
  created_at: string;
  updated_at: string;
}

export interface StickyPaletteMeta {
  color: StickyColor;
  label: string;
  bgHex: string;
  borderHex: string;
  textHex: string;
  dotHex: string;
}

export const STICKY_PALETTES: StickyPaletteMeta[] = [
  {
    color: 'BANANA',
    label: 'Electric Banana',
    bgHex: '#fef08a',
    borderHex: '#facc15',
    textHex: '#713f12',
    dotHex: '#eab308',
  },
  {
    color: 'FLAMINGO',
    label: 'Disco Flamingo',
    bgHex: '#fecdd3',
    borderHex: '#fb7185',
    textHex: '#881337',
    dotHex: '#f43f5e',
  },
  {
    color: 'MINT',
    label: 'Radioactive Mint',
    bgHex: '#a7f3d0',
    borderHex: '#34d399',
    textHex: '#064e3b',
    dotHex: '#10b981',
  },
  {
    color: 'LAGOON',
    label: 'Cyber Lagoon',
    bgHex: '#bae6fd',
    borderHex: '#38bdf8',
    textHex: '#0c4a6e',
    dotHex: '#0ea5e9',
  },
  {
    color: 'LAVENDER',
    label: 'Cosmic Ube',
    bgHex: '#ddd6fe',
    borderHex: '#a78bfa',
    textHex: '#4c1d95',
    dotHex: '#8b5cf6',
  },
  {
    color: 'PEACH',
    label: 'Tangerine Dream',
    bgHex: '#fed7aa',
    borderHex: '#fb923c',
    textHex: '#7c2d12',
    dotHex: '#f97316',
  },
];
