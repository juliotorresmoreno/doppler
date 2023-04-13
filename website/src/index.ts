import { createRoot } from 'react-dom/client';
import Root from './Root';

const container = document.getElementById('app') as HTMLElement
const root = createRoot(container)

const app = Root({})

window.addEventListener('load', function () {
  root.render(app)
})
