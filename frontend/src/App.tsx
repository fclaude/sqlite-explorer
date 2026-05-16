import {useCallback, useRef} from 'react';
import './App.css';
import {DataGrid} from './components/DataGrid';
import {SchemaView} from './components/SchemaView';
import {Sidebar} from './components/Sidebar';
import {SqlEditor} from './components/SqlEditor';
import {StatusBar} from './components/StatusBar';
import {AppProvider, useApp} from './state/AppProvider';
import {MainTab} from './state/types';

const TABS: {id: MainTab; label: string}[] = [
    {id: 'data', label: 'Data'},
    {id: 'schema', label: 'Schema'},
    {id: 'sql', label: 'SQL'},
];

function AppShell() {
    const {state, openDatabase, closeDatabase, selectObject, setTab, setSidebarWidth, toggleGroup, clearError} =
        useApp();
    const dragRef = useRef<{startX: number; startWidth: number} | null>(null);

    const onResizeStart = useCallback(
        (e: React.MouseEvent) => {
            e.preventDefault();
            dragRef.current = {startX: e.clientX, startWidth: state.sidebarWidth};
            const onMove = (ev: MouseEvent) => {
                if (!dragRef.current) return;
                const delta = ev.clientX - dragRef.current.startX;
                setSidebarWidth(dragRef.current.startWidth + delta);
            };
            const onUp = () => {
                dragRef.current = null;
                window.removeEventListener('mousemove', onMove);
                window.removeEventListener('mouseup', onUp);
            };
            window.addEventListener('mousemove', onMove);
            window.addEventListener('mouseup', onUp);
        },
        [state.sidebarWidth, setSidebarWidth],
    );

    const hasDatabase = !!state.dbInfo;

    return (
        <div id="App" className="app-root">
            <header className="app-header">
                <h1 className="app-title">SQLite Explorer</h1>
                <div className="app-header-actions">
                    <button type="button" className="btn" onClick={openDatabase}>
                        Open database
                    </button>
                    <button type="button" className="btn" onClick={closeDatabase} disabled={!hasDatabase}>
                        Close
                    </button>
                </div>
            </header>

            <div className="app-body">
                <div className="sidebar-pane" style={{width: state.sidebarWidth}}>
                    <Sidebar
                        schema={state.schema}
                        selected={state.selected}
                        collapsedGroups={state.collapsedGroups}
                        onSelect={selectObject}
                        onToggleGroup={toggleGroup}
                    />
                </div>
                <div
                    className="resize-handle"
                    role="separator"
                    aria-orientation="vertical"
                    onMouseDown={onResizeStart}
                />
                <main className="main-pane">
                    <div className="tab-bar" role="tablist">
                        {TABS.map((tab) => (
                            <button
                                key={tab.id}
                                type="button"
                                role="tab"
                                aria-selected={state.activeTab === tab.id}
                                className={`tab${state.activeTab === tab.id ? ' active' : ''}`}
                                onClick={() => setTab(tab.id)}
                            >
                                {tab.label}
                            </button>
                        ))}
                    </div>
                    <div className="tab-panel" role="tabpanel">
                        {state.activeTab === 'data' && (
                            <DataGrid
                                selected={state.selected}
                                hasDatabase={hasDatabase}
                                onOpenDatabase={openDatabase}
                            />
                        )}
                        {state.activeTab === 'schema' && (
                            <SchemaView
                                schema={state.schema}
                                selected={state.selected}
                                hasDatabase={hasDatabase}
                                onOpenDatabase={openDatabase}
                            />
                        )}
                        {state.activeTab === 'sql' && (
                            <SqlEditor hasDatabase={hasDatabase} onOpenDatabase={openDatabase} />
                        )}
                    </div>
                </main>
            </div>

            <StatusBar dbInfo={state.dbInfo} status={state.status} onClearError={clearError} />
        </div>
    );
}

function App() {
    return (
        <AppProvider>
            <AppShell />
        </AppProvider>
    );
}

export default App;
