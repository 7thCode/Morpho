export namespace morpho {
	
	export class Morpheme {
	    surface: string;
	    reading?: string;
	    pos: string;
	    pos_detail?: string;
	
	    static createFrom(source: any = {}) {
	        return new Morpheme(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.surface = source["surface"];
	        this.reading = source["reading"];
	        this.pos = source["pos"];
	        this.pos_detail = source["pos_detail"];
	    }
	}

}

