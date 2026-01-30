export namespace main {
	
	export class ConfigData {
	    server_url: string;
	    access_key: string;
	    secret_key: string;
	    auto_connect: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ConfigData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.server_url = source["server_url"];
	        this.access_key = source["access_key"];
	        this.secret_key = source["secret_key"];
	        this.auto_connect = source["auto_connect"];
	    }
	}
	export class ConnectResult {
	    success: boolean;
	    message: string;
	    agent_id: string;
	    agent_name: string;
	
	    static createFrom(source: any = {}) {
	        return new ConnectResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	        this.agent_id = source["agent_id"];
	        this.agent_name = source["agent_name"];
	    }
	}
	export class LogEntry {
	    timestamp: string;
	    level: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new LogEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timestamp = source["timestamp"];
	        this.level = source["level"];
	        this.message = source["message"];
	    }
	}
	export class PermissionInfo {
	    accessibility: boolean;
	    screen_recording: boolean;
	    all_granted: boolean;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new PermissionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.accessibility = source["accessibility"];
	        this.screen_recording = source["screen_recording"];
	        this.all_granted = source["all_granted"];
	        this.message = source["message"];
	    }
	}
	export class StatusInfo {
	    connected: boolean;
	    status: string;
	    agent_id: string;
	    agent_name: string;
	
	    static createFrom(source: any = {}) {
	        return new StatusInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.connected = source["connected"];
	        this.status = source["status"];
	        this.agent_id = source["agent_id"];
	        this.agent_name = source["agent_name"];
	    }
	}
	export class SystemInfo {
	    hostname: string;
	    platform: string;
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new SystemInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hostname = source["hostname"];
	        this.platform = source["platform"];
	        this.version = source["version"];
	    }
	}

}

