import "@vira/shared/src/styles/tokens.css";
import "@vira/shared/src/styles/base.css";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router";
import { ThemeProvider } from "@vira/shared";
import { App } from "./App";
import { SellerAuthProvider } from "./auth";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <ThemeProvider storageKey="vira.seller.theme">
      <BrowserRouter>
        <SellerAuthProvider>
          <App />
        </SellerAuthProvider>
      </BrowserRouter>
    </ThemeProvider>
  </StrictMode>,
);
