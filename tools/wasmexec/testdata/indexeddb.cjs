// A transaction/event fake for Go adapter tests, not a browser storage fallback.
"use strict";

const databases = new Map();
let held = [];
let holdWrites = false;
let failCommit = false;
const later = body => setTimeout(body, 0);
const invoke = (target, name) => { if (typeof target[name] === "function") target[name]({ target }); };

class Transaction {
    constructor(store, mode) {
        this.store = store;
        this.mode = mode;
        this.error = null;
        this.aborted = false;
    }
    objectStore() {
        return {
            get: key => this.operation("get", key),
            put: (value, key) => this.operation("put", key, value)
        };
    }
    operation(kind, key, value) {
        const request = { error: null, result: undefined };
        const data = value?.slice(0);
        const shouldFail = kind === "put" && failCommit;
        if (kind === "put") failCommit = false;
        later(() => {
            request.result = kind === "get" ? this.store.get(key)?.slice(0) : key;
            invoke(request, "onsuccess");
            const finish = () => {
                if (shouldFail || this.aborted) {
                    this.error = { name: "QuotaExceededError", message: "injected failure after put request success" };
                    invoke(this, "onerror");
                    invoke(this, "onabort");
                } else {
                    if (kind === "put") this.store.set(key, data);
                    invoke(this, "oncomplete");
                }
            };
            if (kind === "put" && holdWrites) held.push(finish);
            else later(finish);
        });
        return request;
    }
    abort() { this.aborted = true; }
}

class Database {
    constructor(stores) {
        this.stores = stores;
        this.objectStoreNames = { contains: name => stores.has(name) };
        this.closed = false;
    }
    createObjectStore(name) { this.stores.set(name, new Map()); }
    transaction(name, mode) {
        if (this.closed || !this.stores.has(name)) throw new Error("fake database is closed or missing its object store");
        return new Transaction(this.stores.get(name), mode);
    }
    close() { this.closed = true; }
}

globalThis.indexedDB = {
    open(name) {
        const request = { error: null, result: undefined };
        later(() => {
            const upgrade = !databases.has(name);
            if (upgrade) databases.set(name, new Map());
            request.result = new Database(databases.get(name));
            request.transaction = { abort() { request.error = { name: "AbortError", message: "upgrade aborted" }; } };
            if (upgrade) invoke(request, "onupgradeneeded");
            invoke(request, request.error ? "onerror" : "onsuccess");
        });
        return request;
    }
};
globalThis.th10IndexedDBTest = {
    reset() { databases.clear(); held = []; holdWrites = false; failCommit = false; },
    holdWrites(value) { holdWrites = value; },
    pendingWrites() { return held.length; },
    flushWrites() {
        holdWrites = false;
        const pending = held;
        held = [];
        pending.forEach(finish => later(finish));
    },
    failNextCommit() { failCommit = true; }
};
