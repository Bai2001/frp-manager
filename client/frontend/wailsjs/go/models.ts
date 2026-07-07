export namespace main {
	
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

