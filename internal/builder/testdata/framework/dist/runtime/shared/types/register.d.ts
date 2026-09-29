export interface Register {
}
export type DropFirst<T extends unknown[]> = T extends [unknown, ...infer Rest] ? Rest : never;
export type ViewSend = Record<string, unknown>;
