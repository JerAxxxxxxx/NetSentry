export namespace config {
	
	export class AppConfig {
	    host: string;
	    interval: string;
	    timeout: string;
	    log_dir: string;
	    report_dir: string;
	    summary_every: string;
	    report_every: string;
	
	    static createFrom(source: any = {}) {
	        return new AppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.interval = source["interval"];
	        this.timeout = source["timeout"];
	        this.log_dir = source["log_dir"];
	        this.report_dir = source["report_dir"];
	        this.summary_every = source["summary_every"];
	        this.report_every = source["report_every"];
	    }
	}

}

