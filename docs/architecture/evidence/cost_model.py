"""VIRA infrastructure cost model (doc 17).

Unit prices: ArvanCloud Toman list prices from the third-party machine-readable snapshot
github.com/saqi-sn/arvan-pricing-all, commit c854cbc897b639b7b7ec38115c8fcf33ca89b4c8 (scraped 2026-10-10
from https://www.arvancloud.ir/fa/pricing/all). Planning estimates only — confirm with a vendor quote.
Run: python3 cost_model.py   (prints to stdout; writes nothing)
"""
P = dict(vcore=327_600, ram_gb=255_060, ssd_gb=18_000, local_gb=25_200, ipv4_first=234_000, ipv4_more=468_000,
         cdn_plan={'growth': 3_600_000, 'pro': 36_000_000}, cdn_fair_tb={'growth': 1.5, 'pro': 15}, cdn_gb_over=1_500,
         obj_plan={'growth': 1_800_000, 'pro': 18_000_000}, video_plan={'growth': 2_160_000, 'pro': 21_600_000},
         db={'starter': (264_600, 206_010), 'basic': (579_600, 451_260), 'standard': (819_000, 637_650)}, db_ssd=36_000)
def tiered(x, tiers):
    cost = prev = 0
    for cap, price in tiers:
        if x > prev:
            cost += (min(x, cap) - prev) * price; prev = cap
    return cost
obj_storage = lambda gb: tiered(gb, [(5, 0), (5_000, 3_600), (50_000, 3_060), (500_000, 2_700), (10**12, 2_340)])
video_watch = lambda h: tiered(h, [(50, 0), (20_000, 900), (100_000, 495), (1_000_000, 360), (10**12, 225)])
video_store = lambda h: tiered(h, [(1_000, 18_000), (10_000, 14_400), (100_000, 12_600), (10**12, 10_800)])
def cdn(tb, plan):  # fair use is per day in the price list (50 / 500 GB/day); converted to 30 days
    return P['cdn_plan'][plan] + max(0, tb - P['cdn_fair_tb'][plan]) * 1000 * P['cdn_gb_over']
ips = lambda n: P['ipv4_first'] + max(0, n - 1) * P['ipv4_more']
S = {
 'A': dict(nodes=[('api',3,4,8),('worker',2,4,8),('realtime',2,2,4),('media-img',1,4,8),('postgres',3,8,32),('redis',2,2,8),('nats',3,2,4),('observability',2,4,16)],
           local_gb=1_500, ssd_gb=2_000, staging=0.20, obj_gb=3_000, watch_h=75_000, vstore_h=750, cdn_tb=6, plan='growth', ips=4, offsite_gb=2_000),
 'B': dict(nodes=[('api',8,8,16),('worker',4,8,16),('realtime',3,8,16),('media-img',2,8,16),('postgres',3,32,128),('redis',3,4,32),('nats',3,4,8),('opensearch',3,8,32),('observability',6,8,16)],
           local_gb=6_000, ssd_gb=10_000, staging=0.25, obj_gb=60_000, watch_h=750_000, vstore_h=7_500, cdn_tb=60, plan='pro', ips=8, offsite_gb=20_000),
 'C': dict(nodes=[('api',40,8,16),('worker',16,8,16),('realtime',10,8,16),('media-img',8,16,32),('postgres-money',3,32,128),('postgres-social',3,48,192),('postgres-messaging',3,32,128),('redis',6,8,64),('nats',5,8,16),('opensearch',6,16,64),('observability',10,8,32)],
           local_gb=30_000, ssd_gb=40_000, staging=0.20, obj_gb=600_000, watch_h=7_500_000, vstore_h=60_000, cdn_tb=600, plan='pro', ips=16, offsite_gb=150_000),
}
# B at launch traffic: B infrastructure, Scenario-A traffic volumes
S['B-launch'] = dict(S['B'], obj_gb=3_000, watch_h=75_000, vstore_h=750, cdn_tb=6, offsite_gb=6_000)
def run(s):
    vc = sum(n * c for _, n, c, _ in s['nodes']) * (1 + s['staging'])
    ram = sum(n * r for _, n, _, r in s['nodes']) * (1 + s['staging'])
    L = {
     'Compute (vCore)': vc * P['vcore'], 'Compute (RAM)': ram * P['ram_gb'],
     'Database local disk': s['local_gb'] * P['local_gb'], 'Block storage (SSD)': s['ssd_gb'] * P['ssd_gb'],
     'Public IPv4': ips(s['ips']),
     'Object storage (media) incl. package fee': obj_storage(s['obj_gb']) + P['obj_plan'][s['plan']],
     'Video delivery (watch hours) incl. package fee': video_watch(s['watch_h']) + P['video_plan'][s['plan']],
     'Video storage (stored hours)': video_store(s['vstore_h']),
     'CDN (images, API, static)': cdn(s['cdn_tb'], s['plan']),
     'Off-provider backup copy (2nd provider, assumed same unit price)': obj_storage(s['offsite_gb']),
    }
    return round(vc), round(ram), L
if __name__ == '__main__':
    for k, s in S.items():
        vc, ram, L = run(s); tot = sum(L.values())
        print(f"\n{k}: {vc} vCore, {ram} GB RAM — TOTAL {tot/1e6:,.0f} M Toman/month (list price)")
        for a, b in L.items(): print(f"   {a}: {b/1e6:,.1f} M")
    # Scenario A variant: managed DB (3 nodes x 8 vCore / 32 GB, 500 GB SSD each) instead of self-managed
    sA = S['A']; base = sum(run(sA)[2].values())
    self_db = 3*8*(1+sA['staging'])*P['vcore'] + 3*32*(1+sA['staging'])*P['ram_gb'] + sA['local_gb']*P['local_gb']
    for tier,(vc_p,ram_p) in P['db'].items():
        managed = 3*8*vc_p + 3*32*ram_p + 1_500*P['db_ssd']
        print(f"A with managed DB ({tier}): {(base - self_db + managed)/1e6:,.0f} M (DB {managed/1e6:,.0f} M vs self-managed {self_db/1e6:,.0f} M); backup free only up to 7 days")
    # Video: platform vs self-delivery through CDN at ~450 MB per watch-hour (1 Mbps)
    for k in ('A','B','C'):
        h = S[k]['watch_h']; tb = h * 0.45 / 1000
        print(f"{k}: video via platform {video_watch(h)/1e6:,.0f} M vs self-delivery via CDN overage {tb*1000*P['cdn_gb_over']/1e6:,.0f} M (~{tb:,.0f} TB)")

# v4 (doc 20 §3): Claude-led development — one lightweight staging VM, no production fleet.
def staging(vcore, ram, ssd, obj_gb, label):
    t = vcore * P['vcore'] + ram * P['ram_gb'] + ssd * P['ssd_gb'] + ips(1) + obj_storage(obj_gb)
    print(f"Staging {label}: {vcore} vCore / {ram} GB RAM / {ssd} GB SSD / 1 IPv4 / {obj_gb} GB object storage = {t/1e6:.1f} M Toman/month")
    return t
staging(4, 8, 100, 50, 'small (P1-P2)')
staging(8, 16, 200, 200, 'medium (P3 onward: media/video tests)')
