import {model} from '../../wailsjs/go/models';

/** Plain fixture matching testdata/fixtures.sql for component tests. */
export const fixtureSchemaInfo: model.SchemaInfo = {
    tables: [
        {
            name: 'customers',
            sql: 'CREATE TABLE customers (...)',
            columns: [
                {name: 'id', type: 'INTEGER', notNull: true, primaryKey: 1},
                {name: 'name', type: 'TEXT', notNull: true, primaryKey: 0},
                {name: 'email', type: 'TEXT', notNull: true, primaryKey: 0},
            ],
            foreignKeys: [],
            indexes: [],
        },
        {
            name: 'orders',
            sql: 'CREATE TABLE orders (...)',
            columns: [],
            foreignKeys: [
                {
                    id: 0,
                    seq: 0,
                    from: 'customer_id',
                    to: 'id',
                    table: 'customers',
                    onUpdate: 'NO ACTION',
                    onDelete: 'CASCADE',
                    match: 'NONE',
                },
            ],
            indexes: [],
        },
        {
            name: 'notes',
            sql: 'CREATE TABLE notes (...)',
            columns: [],
            foreignKeys: [],
            indexes: [],
        },
    ],
    views: [
        {
            name: 'orders_by_customer',
            sql: 'CREATE VIEW orders_by_customer AS ...',
            columns: [],
        },
    ],
    indexes: [
        {
            name: 'idx_orders_customer',
            table: 'orders',
            unique: false,
            sql: 'CREATE INDEX idx_orders_customer ON orders(customer_id)',
            columns: [{name: 'customer_id'}],
        },
    ],
    triggers: [
        {
            name: 'notes_updated_at',
            table: 'notes',
            sql: 'CREATE TRIGGER notes_updated_at ...',
        },
    ],
};
