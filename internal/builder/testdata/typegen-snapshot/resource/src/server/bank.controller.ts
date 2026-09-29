import { Server } from '@open-core/framework/server'

@Server.Controller()
export class BankController {
  private balance = 0

  @Server.OnNet('bank:deposit')
  deposit(player: Server.Player, amount: number, note: string) {
    this.balance += amount
  }

  @Server.OnNet('bank:withdraw')
  async withdraw(player: Server.Player, amount: number) {
    this.balance -= amount
  }

  @Server.OnRPC('bank:getBalance')
  async getBalance(player: Server.Player, accountId: string): Promise<number> {
    return this.balance
  }

  @Server.Command({ command: 'balance', description: 'Shows your balance', usage: '/balance' })
  showBalance(player: Server.Player) {}

  @Server.Command({ command: 'reset' })
  reset(player: Server.Player) {}
}
