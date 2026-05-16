import {model} from '../api';

export type MainTab = 'data' | 'schema' | 'sql';

export type ObjectKind = 'table' | 'view' | 'index' | 'trigger';

export interface SelectedObject {
    kind: ObjectKind;
    name: string;
}

export interface StatusState {
    lastError: string | null;
    lastErrorDetail: string | null;
    queryDurationMs: number | null;
    pageInfo: string | null;
    loading: boolean;
}

export interface AppState {
    dbInfo: model.DatabaseInfo | null;
    schema: model.SchemaInfo | null;
    selected: SelectedObject | null;
    activeTab: MainTab;
    sidebarWidth: number;
    collapsedGroups: Record<string, boolean>;
    status: StatusState;
}

export type AppAction =
    | {type: 'SET_LOADING'; loading: boolean}
    | {type: 'SET_ERROR'; message: string; detail?: string}
    | {type: 'CLEAR_ERROR'}
    | {type: 'SET_DB'; dbInfo: model.DatabaseInfo; schema: model.SchemaInfo}
    | {type: 'CLEAR_DB'}
    | {type: 'SET_SCHEMA'; schema: model.SchemaInfo}
    | {type: 'SELECT'; selected: SelectedObject | null }
    | {type: 'SET_TAB'; tab: MainTab}
    | {type: 'SET_SIDEBAR_WIDTH'; width: number}
    | {type: 'TOGGLE_GROUP'; group: string}
    | {type: 'SET_QUERY_DURATION'; ms: number | null}
    | {type: 'SET_PAGE_INFO'; info: string | null};

export const initialState: AppState = {
    dbInfo: null,
    schema: null,
    selected: null,
    activeTab: 'data',
    sidebarWidth: 240,
    collapsedGroups: {},
    status: {
        lastError: null,
        lastErrorDetail: null,
        queryDurationMs: null,
        pageInfo: null,
        loading: false,
    },
};

export function appReducer(state: AppState, action: AppAction): AppState {
    switch (action.type) {
        case 'SET_LOADING':
            return {...state, status: {...state.status, loading: action.loading}};
        case 'SET_ERROR':
            return {
                ...state,
                status: {
                    ...state.status,
                    lastError: action.message,
                    lastErrorDetail: action.detail ?? null,
                    loading: false,
                },
            };
        case 'CLEAR_ERROR':
            return {
                ...state,
                status: {...state.status, lastError: null, lastErrorDetail: null},
            };
        case 'SET_DB':
            return {
                ...state,
                dbInfo: action.dbInfo,
                schema: action.schema,
                selected: null,
                status: {
                    ...state.status,
                    lastError: null,
                    lastErrorDetail: null,
                    queryDurationMs: null,
                    pageInfo: null,
                    loading: false,
                },
            };
        case 'CLEAR_DB':
            return {
                ...initialState,
                sidebarWidth: state.sidebarWidth,
                collapsedGroups: state.collapsedGroups,
            };
        case 'SET_SCHEMA':
            return {...state, schema: action.schema};
        case 'SELECT':
            return {...state, selected: action.selected};
        case 'SET_TAB':
            return {...state, activeTab: action.tab};
        case 'SET_SIDEBAR_WIDTH':
            return {...state, sidebarWidth: action.width};
        case 'TOGGLE_GROUP':
            return {
                ...state,
                collapsedGroups: {
                    ...state.collapsedGroups,
                    [action.group]: !state.collapsedGroups[action.group],
                },
            };
        case 'SET_QUERY_DURATION':
            return {
                ...state,
                status: {...state.status, queryDurationMs: action.ms},
            };
        case 'SET_PAGE_INFO':
            return {
                ...state,
                status: {...state.status, pageInfo: action.info},
            };
        default:
            return state;
    }
}
