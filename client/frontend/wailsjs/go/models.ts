export namespace agent {
	
	export class AllowPortRange {
	    start: number;
	    end: number;
	
	    static createFrom(source: any = {}) {
	        return new AllowPortRange(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.start = source["start"];
	        this.end = source["end"];
	    }
	}
	export class Capabilities {
	    frps_running: boolean;
	    frps_version: string;
	    bind_port: number;
	    allow_ports: AllowPortRange[];
	    support_tcp: boolean;
	    support_udp: boolean;
	    support_http: boolean;
	    support_https: boolean;
	    vhost_http_port: number;
	    vhost_https_port: number;
	    subdomain_host: string;
	    allowed_root_domains: string[];
	
	    static createFrom(source: any = {}) {
	        return new Capabilities(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.frps_running = source["frps_running"];
	        this.frps_version = source["frps_version"];
	        this.bind_port = source["bind_port"];
	        this.allow_ports = this.convertValues(source["allow_ports"], AllowPortRange);
	        this.support_tcp = source["support_tcp"];
	        this.support_udp = source["support_udp"];
	        this.support_http = source["support_http"];
	        this.support_https = source["support_https"];
	        this.vhost_http_port = source["vhost_http_port"];
	        this.vhost_https_port = source["vhost_https_port"];
	        this.subdomain_host = source["subdomain_host"];
	        this.allowed_root_domains = source["allowed_root_domains"];
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

export namespace db {
	
	export class Repo {
	
	
	    static createFrom(source: any = {}) {
	        return new Repo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

export namespace frpc {
	
	export class Manager {
	
	
	    static createFrom(source: any = {}) {
	        return new Manager(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

export namespace main {
	
	export class AddServerInput {
	    name: string;
	    host: string;
	    frps_port: number;
	    frp_token: string;
	    agent_url: string;
	    agent_token: string;
	    is_default: boolean;
	    remark: string;
	
	    static createFrom(source: any = {}) {
	        return new AddServerInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.host = source["host"];
	        this.frps_port = source["frps_port"];
	        this.frp_token = source["frp_token"];
	        this.agent_url = source["agent_url"];
	        this.agent_token = source["agent_token"];
	        this.is_default = source["is_default"];
	        this.remark = source["remark"];
	    }
	}
	export class AddTunnelInput {
	    server_id: string;
	    name: string;
	    protocol: string;
	    local_ip: string;
	    local_port: number;
	    remote_port?: number;
	    custom_domain?: string;
	    subdomain?: string;
	
	    static createFrom(source: any = {}) {
	        return new AddTunnelInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.server_id = source["server_id"];
	        this.name = source["name"];
	        this.protocol = source["protocol"];
	        this.local_ip = source["local_ip"];
	        this.local_port = source["local_port"];
	        this.remote_port = source["remote_port"];
	        this.custom_domain = source["custom_domain"];
	        this.subdomain = source["subdomain"];
	    }
	}
	export class ServerInfo {
	    id: string;
	    name: string;
	    host: string;
	    frps_port: number;
	    frp_token: string;
	    agent_url: string;
	    agent_token: string;
	    is_default: boolean;
	    remark: string;
	
	    static createFrom(source: any = {}) {
	        return new ServerInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.host = source["host"];
	        this.frps_port = source["frps_port"];
	        this.frp_token = source["frp_token"];
	        this.agent_url = source["agent_url"];
	        this.agent_token = source["agent_token"];
	        this.is_default = source["is_default"];
	        this.remark = source["remark"];
	    }
	}
	export class TunnelInfo {
	    id: string;
	    server_id: string;
	    name: string;
	    protocol: string;
	    local_ip: string;
	    local_port: number;
	    remote_port?: number;
	    custom_domain?: string;
	    subdomain?: string;
	    enabled: boolean;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new TunnelInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.server_id = source["server_id"];
	        this.name = source["name"];
	        this.protocol = source["protocol"];
	        this.local_ip = source["local_ip"];
	        this.local_port = source["local_port"];
	        this.remote_port = source["remote_port"];
	        this.custom_domain = source["custom_domain"];
	        this.subdomain = source["subdomain"];
	        this.enabled = source["enabled"];
	        this.status = source["status"];
	    }
	}

}

export namespace sql {
	
	export class DB {
	
	
	    static createFrom(source: any = {}) {
	        return new DB(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

