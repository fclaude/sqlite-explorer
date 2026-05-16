import {render, screen} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {describe, expect, it, vi} from 'vitest';
import {fixtureSchemaInfo} from '../test/fixtureSchema';
import {Sidebar} from './Sidebar';

describe('Sidebar', () => {
    it('renders all fixture object names in the correct groups', () => {
        render(
            <Sidebar
                schema={fixtureSchemaInfo}
                selected={null}
                collapsedGroups={{}}
                onSelect={vi.fn()}
                onToggleGroup={vi.fn()}
            />,
        );

        const names = screen.getAllByText(/./).filter((el) => el.classList.contains('sidebar-item-name'));
        const nameTexts = names.map((el) => el.textContent);
        expect(nameTexts).toContain('customers');
        expect(nameTexts).toContain('orders');
        expect(nameTexts).toContain('notes');
        expect(nameTexts).toContain('orders_by_customer');
        expect(nameTexts).toContain('idx_orders_customer');
        expect(nameTexts).toContain('notes_updated_at');

        expect(screen.getByText('Tables')).toBeInTheDocument();
        expect(screen.getByText('Views')).toBeInTheDocument();
        expect(screen.getByText('Indexes')).toBeInTheDocument();
        expect(screen.getByText('Triggers')).toBeInTheDocument();
    });

    it('calls onSelect when an item is clicked', async () => {
        const user = userEvent.setup();
        const onSelect = vi.fn();
        render(
            <Sidebar
                schema={fixtureSchemaInfo}
                selected={null}
                collapsedGroups={{}}
                onSelect={onSelect}
                onToggleGroup={vi.fn()}
            />,
        );

        await user.click(screen.getByRole('button', {name: 'customers'}));
        expect(onSelect).toHaveBeenCalledWith({kind: 'table', name: 'customers'});
    });
});
