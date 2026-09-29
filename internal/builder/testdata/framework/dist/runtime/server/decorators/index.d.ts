import type { Register } from '../../shared/types/register';
export interface CommandConfig {
    command: string;
    description?: string;
    usage?: string;
}
export declare function OnNet(eventName: string, schema?: unknown): MethodDecorator;
export declare function OnRPC(eventName: string, schema?: unknown): MethodDecorator;
export declare function Command(config: CommandConfig | string, schema?: unknown): MethodDecorator;
export declare function Controller(): ClassDecorator;
export declare function RequiresState(options: { has?: string[]; missing?: string[]; errorMessage?: string }): MethodDecorator;
export declare function Throttle(limit: number, windowMs: number): MethodDecorator;
