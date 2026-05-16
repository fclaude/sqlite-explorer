import React from 'react';
import {model} from '../api';
import {ObjectKind, SelectedObject} from '../state/types';
import './Sidebar.css';

interface SidebarProps {
    schema: model.SchemaInfo | null;
    selected: SelectedObject | null;
    collapsedGroups: Record<string, boolean>;
    onSelect: (item: SelectedObject) => void;
    onToggleGroup: (group: string) => void;
}

const GROUPS: {key: string; label: string; kind: ObjectKind}[] = [
    {key: 'tables', label: 'Tables', kind: 'table'},
    {key: 'views', label: 'Views', kind: 'view'},
    {key: 'indexes', label: 'Indexes', kind: 'index'},
    {key: 'triggers', label: 'Triggers', kind: 'trigger'},
];

export function Sidebar({schema, selected, collapsedGroups, onSelect, onToggleGroup}: SidebarProps) {
    if (!schema) {
        return (
            <aside className="sidebar">
                <p className="sidebar-empty">No schema loaded.</p>
            </aside>
        );
    }

    const itemsByKind: Record<ObjectKind, {name: string; subtitle?: string}[]> = {
        table: schema.tables?.map((t) => ({name: t.name})) ?? [],
        view: schema.views?.map((v) => ({name: v.name})) ?? [],
        index: schema.indexes?.map((i) => ({name: i.name, subtitle: i.table})) ?? [],
        trigger: schema.triggers?.map((t) => ({name: t.name, subtitle: t.table})) ?? [],
    };

    return (
        <aside className="sidebar">
            <nav className="sidebar-nav" aria-label="Database schema">
                {GROUPS.map((group) => {
                    const collapsed = !!collapsedGroups[group.key];
                    const items = itemsByKind[group.kind];
                    return (
                        <section key={group.key} className="sidebar-group">
                            <button
                                type="button"
                                className="sidebar-group-header"
                                onClick={() => onToggleGroup(group.key)}
                                aria-expanded={!collapsed}
                            >
                                <span className="sidebar-chevron">{collapsed ? '>' : 'v'}</span>
                                <span>{group.label}</span>
                                <span className="sidebar-count">{items.length}</span>
                            </button>
                            {!collapsed && (
                                <ul className="sidebar-list">
                                    {items.length === 0 && (
                                        <li className="sidebar-item muted">(none)</li>
                                    )}
                                    {items.map((item) => {
                                        const isSelected =
                                            selected?.kind === group.kind && selected.name === item.name;
                                        return (
                                            <li key={item.name}>
                                                <button
                                                    type="button"
                                                    className={`sidebar-item${isSelected ? ' selected' : ''}`}
                                                    onClick={() => onSelect({kind: group.kind, name: item.name})}
                                                >
                                                    <span className="sidebar-item-name">{item.name}</span>
                                                    {item.subtitle && (
                                                        <span className="sidebar-item-sub">{item.subtitle}</span>
                                                    )}
                                                </button>
                                            </li>
                                        );
                                    })}
                                </ul>
                            )}
                        </section>
                    );
                })}
            </nav>
        </aside>
    );
}
