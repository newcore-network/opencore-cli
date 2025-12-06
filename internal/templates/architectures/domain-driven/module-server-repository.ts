import { Server } from '@open-core/framework';

@Server.Injectable()
export class {{.ModuleNamePascal}}Repository {
  private storage = new Map<number, any>();

  async findByPlayer(playerId: number): Promise<any> {
    return this.storage.get(playerId) || null;
  }

  async save(playerId: number, data: any): Promise<boolean> {
    this.storage.set(playerId, data);
    return true;
  }

  async delete(playerId: number): Promise<boolean> {
    return this.storage.delete(playerId);
  }
}

