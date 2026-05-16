export namespace model {
	
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

}

