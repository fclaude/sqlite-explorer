import {createContext, ReactNode, useCallback, useContext, useMemo, useReducer} from 'react';
import {formatAPIError, WailsAPI} from '../api';
import {appReducer, AppState, initialState, SelectedObject} from './types';

interface AppContextValue {
    state: AppState;
    openDatabase: () => Promise<void>;
    closeDatabase: () => Promise<void>;
    selectObject: (selected: SelectedObject | null) => void;
    setTab: (tab: AppState['activeTab']) => void;
    setSidebarWidth: (width: number) => void;
    toggleGroup: (group: string) => void;
    clearError: () => void;
}

const AppContext = createContext<AppContextValue | null>(null);

export function AppProvider({children}: {children: ReactNode}) {
    const [state, dispatch] = useReducer(appReducer, initialState);

    const openDatabase = useCallback(async () => {
        dispatch({type: 'SET_LOADING', loading: true});
        dispatch({type: 'CLEAR_ERROR'});
        try {
            const dbInfo = await WailsAPI.openDatabase();
            const schema = await WailsAPI.getSchema();
            dispatch({type: 'SET_DB', dbInfo, schema});
        } catch (err) {
            const {message, detail} = formatAPIError(err);
            dispatch({type: 'SET_ERROR', message, detail});
        }
    }, []);

    const closeDatabase = useCallback(async () => {
        dispatch({type: 'SET_LOADING', loading: true});
        try {
            await WailsAPI.closeDatabase();
            dispatch({type: 'CLEAR_DB'});
        } catch (err) {
            const {message, detail} = formatAPIError(err);
            dispatch({type: 'SET_ERROR', message, detail});
        }
    }, []);

    const selectObject = useCallback((selected: SelectedObject | null) => {
        dispatch({type: 'SELECT', selected});
        if (selected?.kind === 'table' || selected?.kind === 'view') {
            dispatch({type: 'SET_TAB', tab: 'data'});
        } else if (selected) {
            dispatch({type: 'SET_TAB', tab: 'schema'});
        }
    }, []);

    const setTab = useCallback((tab: AppState['activeTab']) => {
        dispatch({type: 'SET_TAB', tab});
    }, []);

    const setSidebarWidth = useCallback((width: number) => {
        dispatch({type: 'SET_SIDEBAR_WIDTH', width: Math.max(180, Math.min(480, width))});
    }, []);

    const toggleGroup = useCallback((group: string) => {
        dispatch({type: 'TOGGLE_GROUP', group});
    }, []);

    const clearError = useCallback(() => {
        dispatch({type: 'CLEAR_ERROR'});
    }, []);

    const value = useMemo(
        () => ({
            state,
            openDatabase,
            closeDatabase,
            selectObject,
            setTab,
            setSidebarWidth,
            toggleGroup,
            clearError,
        }),
        [state, openDatabase, closeDatabase, selectObject, setTab, setSidebarWidth, toggleGroup, clearError],
    );

    return <AppContext.Provider value={value}>{children}</AppContext.Provider>;
}

export function useApp(): AppContextValue {
    const ctx = useContext(AppContext);
    if (!ctx) {
        throw new Error('useApp must be used within AppProvider');
    }
    return ctx;
}
