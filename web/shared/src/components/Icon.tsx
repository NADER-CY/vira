// Material Symbols (Apache-2.0), bundled as SVG at build time — no CDN at runtime (DS-05, DS-06).
import accountBalanceWallet from "@material-symbols/svg-400/outlined/account_balance_wallet.svg?raw";
import adminPanelSettings from "@material-symbols/svg-400/outlined/admin_panel_settings.svg?raw";
import assignmentReturn from "@material-symbols/svg-400/outlined/assignment_return.svg?raw";
import block from "@material-symbols/svg-400/outlined/block.svg?raw";
import campaign from "@material-symbols/svg-400/outlined/campaign.svg?raw";
import chat from "@material-symbols/svg-400/outlined/chat.svg?raw";
import chevronLeft from "@material-symbols/svg-400/outlined/chevron_left.svg?raw";
import darkMode from "@material-symbols/svg-400/outlined/dark_mode.svg?raw";
import dashboard from "@material-symbols/svg-400/outlined/dashboard.svg?raw";
import errorIcon from "@material-symbols/svg-400/outlined/error.svg?raw";
import flag from "@material-symbols/svg-400/outlined/flag.svg?raw";
import group from "@material-symbols/svg-400/outlined/group.svg?raw";
import history from "@material-symbols/svg-400/outlined/history.svg?raw";
import inbox from "@material-symbols/svg-400/outlined/inbox.svg?raw";
import inventory2 from "@material-symbols/svg-400/outlined/inventory_2.svg?raw";
import lightMode from "@material-symbols/svg-400/outlined/light_mode.svg?raw";
import localShipping from "@material-symbols/svg-400/outlined/local_shipping.svg?raw";
import lock from "@material-symbols/svg-400/outlined/lock.svg?raw";
import logout from "@material-symbols/svg-400/outlined/logout.svg?raw";
import monitoring from "@material-symbols/svg-400/outlined/monitoring.svg?raw";
import notifications from "@material-symbols/svg-400/outlined/notifications.svg?raw";
import payments from "@material-symbols/svg-400/outlined/payments.svg?raw";
import receiptLong from "@material-symbols/svg-400/outlined/receipt_long.svg?raw";
import refresh from "@material-symbols/svg-400/outlined/refresh.svg?raw";
import search from "@material-symbols/svg-400/outlined/search.svg?raw";
import sell from "@material-symbols/svg-400/outlined/sell.svg?raw";
import settings from "@material-symbols/svg-400/outlined/settings.svg?raw";
import shieldLock from "@material-symbols/svg-400/outlined/shield_lock.svg?raw";
import shoppingBag from "@material-symbols/svg-400/outlined/shopping_bag.svg?raw";
import smartToy from "@material-symbols/svg-400/outlined/smart_toy.svg?raw";
import storefront from "@material-symbols/svg-400/outlined/storefront.svg?raw";
import verifiedUser from "@material-symbols/svg-400/outlined/verified_user.svg?raw";
import videoLibrary from "@material-symbols/svg-400/outlined/video_library.svg?raw";
import checkCircle from "@material-symbols/svg-400/outlined/check_circle.svg?raw";
import chevronRight from "@material-symbols/svg-400/outlined/chevron_right.svg?raw";
import close from "@material-symbols/svg-400/outlined/close.svg?raw";
import devices from "@material-symbols/svg-400/outlined/devices.svg?raw";
import filterAltOff from "@material-symbols/svg-400/outlined/filter_alt_off.svg?raw";
import gavel from "@material-symbols/svg-400/outlined/gavel.svg?raw";
import howToReg from "@material-symbols/svg-400/outlined/how_to_reg.svg?raw";
import person from "@material-symbols/svg-400/outlined/person.svg?raw";
import personOff from "@material-symbols/svg-400/outlined/person_off.svg?raw";
import report from "@material-symbols/svg-400/outlined/report.svg?raw";
import visibility from "@material-symbols/svg-400/outlined/visibility.svg?raw";
import warning from "@material-symbols/svg-400/outlined/warning.svg?raw";

const icons = {
  account_balance_wallet: accountBalanceWallet, admin_panel_settings: adminPanelSettings, assignment_return: assignmentReturn,
  block, campaign, chat, chevron_left: chevronLeft, dark_mode: darkMode, dashboard, error: errorIcon, flag, group, history,
  inbox, inventory_2: inventory2, light_mode: lightMode, local_shipping: localShipping, lock, logout, monitoring,
  notifications, payments, receipt_long: receiptLong, refresh, search, sell, settings, shield_lock: shieldLock,
  shopping_bag: shoppingBag, smart_toy: smartToy, storefront, verified_user: verifiedUser, video_library: videoLibrary,
  check_circle: checkCircle, chevron_right: chevronRight, close, devices, filter_alt_off: filterAltOff, gavel, how_to_reg: howToReg, person, person_off: personOff, report, visibility, warning,
} as const;

export type IconName = keyof typeof icons;

/** Icons whose meaning is directional are mirrored in RTL (DS-06). */
const MIRROR: ReadonlySet<IconName> = new Set<IconName>(["chevron_left", "chevron_right", "logout", "assignment_return"]);

export function Icon({ name, label, className }: { name: IconName; label?: string; className?: string }) {
  return (
    <span
      className={["v-icon", MIRROR.has(name) ? "v-icon--mirror" : "", className ?? ""].join(" ").trim()}
      role={label ? "img" : undefined}
      aria-label={label}
      aria-hidden={label ? undefined : true}
      dangerouslySetInnerHTML={{ __html: icons[name] }}
    />
  );
}
