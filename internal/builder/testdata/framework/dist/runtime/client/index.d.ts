import { Controller, OnNet, OnRPC, OnView } from './decorators';
export * from './decorators';
export * from './webview-bridge';
export interface ClientApi {
    OnNet: typeof OnNet;
    OnRPC: typeof OnRPC;
    OnView: typeof OnView;
    Controller: typeof Controller;
}
export declare const Client: ClientApi;
