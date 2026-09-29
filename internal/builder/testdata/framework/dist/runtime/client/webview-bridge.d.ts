import type { ViewSend } from '../shared/types/register';
export declare class WebViewBridge<TSend extends object = ViewSend> {
    send<K extends keyof TSend & string>(action: K, data: TSend[K]): void;
    sendRaw(action: string, data: unknown): void;
}
export declare const WebView: WebViewBridge;
export declare const NUI: WebViewBridge;
export declare function createWebView<TSend extends object = ViewSend>(viewId: string): WebViewBridge<TSend>;
