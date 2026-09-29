import { Command, Controller, OnNet, OnRPC, RequiresState, Throttle } from './decorators';
export * from './decorators';
export interface Player {
    clientID: number;
    emit(eventName: string, ...args: unknown[]): void;
}
export interface ServerApi {
    OnNet: typeof OnNet;
    OnRPC: typeof OnRPC;
    Command: typeof Command;
    Controller: typeof Controller;
    RequiresState: typeof RequiresState;
    Throttle: typeof Throttle;
}
export declare const Server: ServerApi;
export declare namespace Server {
    type Player = import('./index').Player;
}
