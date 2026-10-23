import { Command, Controller, OnLibraryEvent, OnNet, OnRPC, RequiresState, Throttle } from './decorators';
export * from './decorators';
export interface Player {
    clientID: number;
    emit(eventName: string, ...args: unknown[]): void;
}
export interface ServerApi {
    OnNet: typeof OnNet;
    OnRPC: typeof OnRPC;
    Command: typeof Command;
    OnLibraryEvent: typeof OnLibraryEvent;
    Controller: typeof Controller;
    RequiresState: typeof RequiresState;
    Throttle: typeof Throttle;
}
export declare const Server: ServerApi;
export declare namespace Server {
    type Player = import('./index').Player;
}
export declare function createServerLibrary<const N extends string = string>(name: N): import('../core/library/types').OpenCoreServerLibrary<N>;
