export namespace main {
	
	export class Stats {
	    word_count: number;
	    is_trained: boolean;
	    pos_tags: string[];
	
	    static createFrom(source: any = {}) {
	        return new Stats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.word_count = source["word_count"];
	        this.is_trained = source["is_trained"];
	        this.pos_tags = source["pos_tags"];
	    }
	}

}

export namespace morpho {
	
	export class DictEntry {
	    surface: string;
	    reading?: string;
	    pos: string;
	    pos_detail?: string;
	    freq: number;
	
	    static createFrom(source: any = {}) {
	        return new DictEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.surface = source["surface"];
	        this.reading = source["reading"];
	        this.pos = source["pos"];
	        this.pos_detail = source["pos_detail"];
	        this.freq = source["freq"];
	    }
	}
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

