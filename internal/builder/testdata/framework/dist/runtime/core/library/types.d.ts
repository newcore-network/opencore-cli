export interface LibraryBus {
    on<TPayload = unknown>(event: string, handler: (payload?: TPayload) => void): void;
    emit<TPayload = unknown>(event: string, payload?: TPayload): void;
}
export interface OpenCoreLibraryBase<N extends string = string> extends LibraryBus {
    name: N;
}
export interface OpenCoreServerLibrary<N extends string = string> extends OpenCoreLibraryBase<N> {
    side: 'server';
}
export interface OpenCoreClientLibrary<N extends string = string> extends OpenCoreLibraryBase<N> {
    side: 'client';
}
