import React from 'react';
import ReactDOM from 'react-dom/client';
import { App } from './App';
import { initTheme } from './theme';
import { applyProwlarrLayoutCssVars } from './prowlarrLayoutConstants';
import './styles.css';

initTheme();
applyProwlarrLayoutCssVars();

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);

