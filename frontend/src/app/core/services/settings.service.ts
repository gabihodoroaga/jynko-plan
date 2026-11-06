import { Injectable, computed, inject, signal } from '@angular/core';
import { ProjectService } from './project.service';
import { ALL_STATUSES, ProjectSettingsModel, WorkItemStatus } from '../models/project.model';

export const DEFAULT_VISIBLE_BOARD_COLUMNS: WorkItemStatus[] = [
  'BACKLOG',
  'TODO',
  'IN_PROGRESS',
  'IN_REVIEW',
  'DONE',
];

export interface GeneralAppSettings {
  instanceName: string;
  defaultLandingRoute: string;
  defaultWorkItemView: string;
  compactTableRows: boolean;
  showSubItemProgressOnCards: boolean;
  enableKeyboardShortcuts: boolean;
  dailyMorningDigest: boolean;
  notifyOnAssignment: boolean;
  notifyOnWikiUnlock: boolean;
}

export interface ProjectSpecificSettings {
  projectId: string;
  defaultNewItemStatus: string;
  defaultNewItemPriority: string;
  estimationScale: string;
  cycleDurationWeeks: string;
  visibleBoardColumns: WorkItemStatus[];
  autoAssignToLead: boolean;
  autoRolloverUnfinishedCycleItems: boolean;
  requireEstimateBeforeInProgress: boolean;
  autoLockPublishedRunbooks: boolean;
  createVersionOnEverySave: boolean;
}

export const DEFAULT_PROJECT_SETTINGS = (projectId: string): ProjectSpecificSettings => ({
  projectId,
  defaultNewItemStatus: 'TODO',
  defaultNewItemPriority: 'MEDIUM',
  estimationScale: 'FIBONACCI',
  cycleDurationWeeks: '2',
  visibleBoardColumns: [...DEFAULT_VISIBLE_BOARD_COLUMNS],
  autoAssignToLead: false,
  autoRolloverUnfinishedCycleItems: true,
  requireEstimateBeforeInProgress: false,
  autoLockPublishedRunbooks: false,
  createVersionOnEverySave: true,
});

export function toModelSettings(s: ProjectSpecificSettings): ProjectSettingsModel {
  return {
    default_new_item_status: s.defaultNewItemStatus,
    default_new_item_priority: s.defaultNewItemPriority,
    estimation_scale: s.estimationScale,
    cycle_duration_weeks: s.cycleDurationWeeks,
    visible_board_columns: s.visibleBoardColumns,
    auto_assign_to_lead: s.autoAssignToLead,
    auto_rollover_unfinished_cycle_items: s.autoRolloverUnfinishedCycleItems,
    require_estimate_before_in_progress: s.requireEstimateBeforeInProgress,
    auto_lock_published_runbooks: s.autoLockPublishedRunbooks,
    create_version_on_every_save: s.createVersionOnEverySave,
  };
}

export function fromModelSettings(projectId: string, m?: ProjectSettingsModel): ProjectSpecificSettings {
  const d = DEFAULT_PROJECT_SETTINGS(projectId);
  if (!m) return d;
  return {
    projectId,
    defaultNewItemStatus: m.default_new_item_status || d.defaultNewItemStatus,
    defaultNewItemPriority: m.default_new_item_priority || d.defaultNewItemPriority,
    estimationScale: m.estimation_scale || d.estimationScale,
    cycleDurationWeeks: m.cycle_duration_weeks || d.cycleDurationWeeks,
    visibleBoardColumns:
      m.visible_board_columns && m.visible_board_columns.length > 0
        ? m.visible_board_columns
        : d.visibleBoardColumns,
    autoAssignToLead: m.auto_assign_to_lead ?? d.autoAssignToLead,
    autoRolloverUnfinishedCycleItems:
      m.auto_rollover_unfinished_cycle_items ?? d.autoRolloverUnfinishedCycleItems,
    requireEstimateBeforeInProgress:
      m.require_estimate_before_in_progress ?? d.requireEstimateBeforeInProgress,
    autoLockPublishedRunbooks: m.auto_lock_published_runbooks ?? d.autoLockPublishedRunbooks,
    createVersionOnEverySave: m.create_version_on_every_save ?? d.createVersionOnEverySave,
  };
}

const LOCAL_STORAGE_GENERAL_KEY = 'jynko_general_settings';

function loadInitialGeneralSettings(): GeneralAppSettings {
  const defaults: GeneralAppSettings = {
    instanceName: 'Jynko Engineering',
    defaultLandingRoute: '/home',
    defaultWorkItemView: 'LIST',
    compactTableRows: true,
    showSubItemProgressOnCards: true,
    enableKeyboardShortcuts: true,
    dailyMorningDigest: true,
    notifyOnAssignment: true,
    notifyOnWikiUnlock: false,
  };
  try {
    const raw = localStorage.getItem(LOCAL_STORAGE_GENERAL_KEY);
    if (raw) {
      return { ...defaults, ...JSON.parse(raw) };
    }
  } catch {}
  return defaults;
}

@Injectable({
  providedIn: 'root',
})
export class SettingsService {
  private readonly projectService = inject(ProjectService);

  readonly activeScope = signal<'GENERAL' | 'PROJECT'>('GENERAL');

  readonly generalSettings = signal<GeneralAppSettings>(loadInitialGeneralSettings());

  readonly projectSettingsByProject = signal<Record<string, ProjectSpecificSettings>>({});

  readonly activeProjectSettings = computed<ProjectSpecificSettings>(() => {
    const project = this.projectService.activeProject();
    if (!project) {
      const pid = this.projectService.activeProjectId();
      return this.projectSettingsByProject()[pid] ?? DEFAULT_PROJECT_SETTINGS(pid);
    }
    const local = this.projectSettingsByProject()[project.id];
    if (local) {
      return local;
    }
    return fromModelSettings(project.id, project.settings);
  });

  readonly visibleBoardStatuses = computed<WorkItemStatus[]>(() => {
    return this.activeProjectSettings().visibleBoardColumns;
  });

  isBoardColumnVisible(status: WorkItemStatus): boolean {
    return this.visibleBoardStatuses().includes(status);
  }

  toggleBoardColumn(status: WorkItemStatus): void {
    const current = this.visibleBoardStatuses();
    let next: WorkItemStatus[];
    if (current.includes(status)) {
      if (current.length <= 1) {
        return; // Keep at least 1 column visible on the board
      }
      next = current.filter((s) => s !== status);
    } else {
      const targetSet = new Set([...current, status]);
      next = ALL_STATUSES.map((s) => s.value).filter((s) => targetSet.has(s));
    }
    this.updateActiveProjectSetting('visibleBoardColumns', next);
  }

  showAllBoardColumns(): void {
    this.updateActiveProjectSetting(
      'visibleBoardColumns',
      ALL_STATUSES.map((s) => s.value)
    );
  }

  resetBoardColumnsToDefault(): void {
    this.updateActiveProjectSetting('visibleBoardColumns', [...DEFAULT_VISIBLE_BOARD_COLUMNS]);
  }

  openGeneralSettings(): void {
    this.activeScope.set('GENERAL');
  }

  openProjectSettings(projectId?: string): void {
    if (projectId && projectId !== this.projectService.activeProjectId()) {
      this.projectService.selectProject(projectId);
    }
    this.activeScope.set('PROJECT');
  }

  updateGeneral<K extends keyof GeneralAppSettings>(key: K, value: GeneralAppSettings[K]): void {
    this.generalSettings.update((prev) => {
      const next = { ...prev, [key]: value };
      try {
        localStorage.setItem(LOCAL_STORAGE_GENERAL_KEY, JSON.stringify(next));
      } catch {}
      return next;
    });
  }

  updateActiveProjectSetting<K extends keyof ProjectSpecificSettings>(
    key: K,
    value: ProjectSpecificSettings[K]
  ): void {
    const projectId = this.projectService.activeProjectId();
    if (!projectId) return;
    const current = this.activeProjectSettings();
    const updated: ProjectSpecificSettings = {
      ...current,
      [key]: value,
    };
    this.projectSettingsByProject.update((map) => ({
      ...map,
      [projectId]: updated,
    }));
    const payloadSettings = toModelSettings(updated);
    this.projectService.updateProject(projectId, { settings: payloadSettings });
  }
}
