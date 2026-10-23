import { Controller, OnLibraryEvent, OnNet, OnRPC, OnView } from './decorators';
export * from './decorators';
export * from './webview-bridge';
export interface ClientApi {
    OnNet: typeof OnNet;
    OnRPC: typeof OnRPC;
    OnView: typeof OnView;
    OnLibraryEvent: typeof OnLibraryEvent;
    Controller: typeof Controller;
    createClientLibrary: typeof createClientLibrary;
}
export declare const Client: ClientApi;
export declare function createClientLibrary<const N extends string = string>(name: N): import('../core/library/types').OpenCoreClientLibrary<N>;
