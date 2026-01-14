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
	export class InvoiceItem {
	    codigo: string;
	    descricao: string;
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new InvoiceItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.codigo = source["codigo"];
	        this.descricao = source["descricao"];
	        this.total = source["total"];
	    }
	}
	export class InvoiceTomador {
	    nome: string;
	    cpfCnpj: string;
	    ie: string;
	    telefone: string;
	    endereco: string;
	    cidade: string;
	    uf: string;
	
	    static createFrom(source: any = {}) {
	        return new InvoiceTomador(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nome = source["nome"];
	        this.cpfCnpj = source["cpfCnpj"];
	        this.ie = source["ie"];
	        this.telefone = source["telefone"];
	        this.endereco = source["endereco"];
	        this.cidade = source["cidade"];
	        this.uf = source["uf"];
	    }
	}
	export class InvoiceEmitter {
	    nome: string;
	    fantasia: string;
	    cpfCnpj: string;
	    ie: string;
	    endereco: string;
	    cidade: string;
	    estado: string;
	    telefone: string;
	    logoBase64: string;
	
	    static createFrom(source: any = {}) {
	        return new InvoiceEmitter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nome = source["nome"];
	        this.fantasia = source["fantasia"];
	        this.cpfCnpj = source["cpfCnpj"];
	        this.ie = source["ie"];
	        this.endereco = source["endereco"];
	        this.cidade = source["cidade"];
	        this.estado = source["estado"];
	        this.telefone = source["telefone"];
	        this.logoBase64 = source["logoBase64"];
	    }
	}
	export class InvoiceData {
	    id: string;
	    numero: number;
	    // Go type: time
	    dataEmissao: any;
	    emitente: InvoiceEmitter;
	    tomador: InvoiceTomador;
	    itens: InvoiceItem[];
	    totalDescontoAplicado: number;
	    totalOutrasDespesas: number;
	    total: number;
	    observacao: string;
	    formaPagamento: string;
	
	    static createFrom(source: any = {}) {
	        return new InvoiceData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.numero = source["numero"];
	        this.dataEmissao = this.convertValues(source["dataEmissao"], null);
	        this.emitente = this.convertValues(source["emitente"], InvoiceEmitter);
	        this.tomador = this.convertValues(source["tomador"], InvoiceTomador);
	        this.itens = this.convertValues(source["itens"], InvoiceItem);
	        this.totalDescontoAplicado = source["totalDescontoAplicado"];
	        this.totalOutrasDespesas = source["totalOutrasDespesas"];
	        this.total = source["total"];
	        this.observacao = source["observacao"];
	        this.formaPagamento = source["formaPagamento"];
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
	export class InvoiceDetails {
	    id: string;
	    numero: number;
	    serie: string;
	    tipo: string;
	    chave: string;
	    situacao: string;
	    emitente: string;
	    tomador: string;
	    // Go type: time
	    data: any;
	    valor: number;
	
	    static createFrom(source: any = {}) {
	        return new InvoiceDetails(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.numero = source["numero"];
	        this.serie = source["serie"];
	        this.tipo = source["tipo"];
	        this.chave = source["chave"];
	        this.situacao = source["situacao"];
	        this.emitente = source["emitente"];
	        this.tomador = source["tomador"];
	        this.data = this.convertValues(source["data"], null);
	        this.valor = source["valor"];
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
	
	
	export class InvoiceSummary {
	    id: string;
	    numero: number;
	    // Go type: time
	    dataEmissao: any;
	    tomadorNome: string;
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new InvoiceSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.numero = source["numero"];
	        this.dataEmissao = this.convertValues(source["dataEmissao"], null);
	        this.tomadorNome = source["tomadorNome"];
	        this.total = source["total"];
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

