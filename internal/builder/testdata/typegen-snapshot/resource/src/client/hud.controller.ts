import { Client, WebView } from '@open-core/framework/client'

export interface HudState {
  cash: number
  visible: boolean
}

@Client.Controller()
export class HudController {
  @Client.OnNet('hud:setCash')
  setCash(amount: number) {}

  @Client.OnRPC('bank:confirm')
  async confirm(message: string): Promise<boolean> {
    return true
  }

  pushState(state: HudState) {
    WebView.send('hud:state', state)
  }

  clear() {
    WebView.send('hud:clear', undefined)
  }

  @Client.OnView('hud:close')
  close(data: { reason: string }) {}

  @Client.OnView('hud:ready')
  ready() {}
}
