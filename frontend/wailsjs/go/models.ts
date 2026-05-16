export namespace model {
	
	export class CellValue {
	    kind: string;
	    value: any;
	
	    static createFrom(source: any = {}) {
	        return new CellValue(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.value = source["value"];
	    }
	}
	export class ColumnInfo {
	    name: string;
	    type: string;
	    notNull: boolean;
	    defaultValue?: string;
	    primaryKey: number;
	
	    static createFrom(source: any = {}) {
	        return new ColumnInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.notNull = source["notNull"];
	        this.defaultValue = source["defaultValue"];
	        this.primaryKey = source["primaryKey"];
	    }
	}
	export class ColumnResult {
	    name: string;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new ColumnResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	    }
	}
	export class DatabaseInfo {
	    path: string;
	    sizeBytes: number;
	    readOnly: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DatabaseInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.sizeBytes = source["sizeBytes"];
	        this.readOnly = source["readOnly"];
	    }
	}
	export class ForeignKeyInfo {
	    id: number;
	    seq: number;
	    from: string;
	    to: string;
	    table: string;
	    onUpdate: string;
	    onDelete: string;
	    match: string;
	
	    static createFrom(source: any = {}) {
	        return new ForeignKeyInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.seq = source["seq"];
	        this.from = source["from"];
	        this.to = source["to"];
	        this.table = source["table"];
	        this.onUpdate = source["onUpdate"];
	        this.onDelete = source["onDelete"];
	        this.match = source["match"];
	    }
	}
	export class IndexColumnInfo {
	    name: string;
	    collSeq?: string;
	
	    static createFrom(source: any = {}) {
	        return new IndexColumnInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.collSeq = source["collSeq"];
	    }
	}
	export class IndexInfo {
	    name: string;
	    table: string;
	    unique: boolean;
	    sql: string;
	    columns: IndexColumnInfo[];
	
	    static createFrom(source: any = {}) {
	        return new IndexInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.table = source["table"];
	        this.unique = source["unique"];
	        this.sql = source["sql"];
	        this.columns = this.convertValues(source["columns"], IndexColumnInfo);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TriggerInfo {
	    name: string;
	    table: string;
	    sql: string;
	
	    static createFrom(source: any = {}) {
	        return new TriggerInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.table = source["table"];
	        this.sql = source["sql"];
	    }
	}
	export class ViewInfo {
	    name: string;
	    sql: string;
	    columns: ColumnInfo[];
	
	    static createFrom(source: any = {}) {
	        return new ViewInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.sql = source["sql"];
	        this.columns = this.convertValues(source["columns"], ColumnInfo);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TableIndexRef {
	    name: string;
	    unique: boolean;
	    origin: string;
	    partial: boolean;
	    columns: IndexColumnInfo[];
	
	    static createFrom(source: any = {}) {
	        return new TableIndexRef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.unique = source["unique"];
	        this.origin = source["origin"];
	        this.partial = source["partial"];
	        this.columns = this.convertValues(source["columns"], IndexColumnInfo);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TableInfo {
	    name: string;
	    sql: string;
	    columns: ColumnInfo[];
	    foreignKeys: ForeignKeyInfo[];
	    indexes: TableIndexRef[];
	
	    static createFrom(source: any = {}) {
	        return new TableInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.sql = source["sql"];
	        this.columns = this.convertValues(source["columns"], ColumnInfo);
	        this.foreignKeys = this.convertValues(source["foreignKeys"], ForeignKeyInfo);
	        this.indexes = this.convertValues(source["indexes"], TableIndexRef);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SchemaInfo {
	    tables: TableInfo[];
	    views: ViewInfo[];
	    indexes: IndexInfo[];
	    triggers: TriggerInfo[];
	
	    static createFrom(source: any = {}) {
	        return new SchemaInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tables = this.convertValues(source["tables"], TableInfo);
	        this.views = this.convertValues(source["views"], ViewInfo);
	        this.indexes = this.convertValues(source["indexes"], IndexInfo);
	        this.triggers = this.convertValues(source["triggers"], TriggerInfo);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class TableRowsRequest {
	    table: string;
	    pageSize: number;
	    page: number;
	    sortColumn: string;
	    sortDesc: boolean;
	    filter: string;
	    withTotal: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TableRowsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.table = source["table"];
	        this.pageSize = source["pageSize"];
	        this.page = source["page"];
	        this.sortColumn = source["sortColumn"];
	        this.sortDesc = source["sortDesc"];
	        this.filter = source["filter"];
	        this.withTotal = source["withTotal"];
	    }
	}
	export class TableRowsResponse {
	    columns: ColumnResult[];
	    rows: CellValue[][];
	    page: number;
	    pageSize: number;
	    totalRows?: number;
	    durationMs: number;
	
	    static createFrom(source: any = {}) {
	        return new TableRowsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.columns = this.convertValues(source["columns"], ColumnResult);
	        this.rows = this.convertValues(source["rows"], CellValue);
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
	        this.totalRows = source["totalRows"];
	        this.durationMs = source["durationMs"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	

}

