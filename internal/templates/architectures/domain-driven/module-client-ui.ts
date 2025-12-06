import { Client } from '@open-core/framework';

@Client.Injectable()
export class {{.ModuleNamePascal}}UI {
  private isVisible = false;

  show() {
    this.isVisible = true;
    // Show NUI or UI logic
    console.log('{{.ModuleNamePascal}} UI shown');
  }

  hide() {
    this.isVisible = false;
    // Hide NUI or UI logic
    console.log('{{.ModuleNamePascal}} UI hidden');
  }

  update(data: any) {
    // Update UI with new data
    console.log('UI updated with:', data);
  }
}

