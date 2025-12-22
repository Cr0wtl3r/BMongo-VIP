export namespace operations {
	
	export class BackupResult {
	    path: string;
	    size: number;
	    timestamp: string;
	
	    static createFrom(source: any = {}) {
	        return new BackupResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.size = source["size"];
	        this.timestamp = source["timestamp"];
	    }
	}
	export class InvoiceDetails {
	    id: string;
	    numero: string;
	    serie: string;
	    chave: string;
	    data: string;
	    cliente: string;
	    valor: number;
	    situacao: string;
	
	    static createFrom(source: any = {}) {
	        return new InvoiceDetails(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.numero = source["numero"];
	        this.serie = source["serie"];
	        this.chave = source["chave"];
	        this.data = source["data"];
	        this.cliente = source["cliente"];
	        this.valor = source["valor"];
	        this.situacao = source["situacao"];
	    }
	}

}

export namespace windows {
	
	export class DigiProcess {
	    name: string;
	    pid: number;
	
	    static createFrom(source: any = {}) {
	        return new DigiProcess(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.pid = source["pid"];
	    }
	}
	export class DigiService {
	    name: string;
	    displayName: string;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new DigiService(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.status = source["status"];
	    }
	}

}

