#include <linux/bpf.h>
#include <linux/ip.h>
#include <linux/icmp.h>

SEC("tc")
int block_icmp_egress(struct __sk_buff *skb) {
    void *data = (void *)(long)skb->data;
    void *data_end = (void *)(long)skb->data_end;
    
    if (data + sizeof(struct iphdr) > data_end)
        return TC_ACT_OK;
    
    struct iphdr *iph = (struct iphdr *)data;
    
    if (iph->version != 4 || iph->protocol != IPPROTO_ICMP)
        return TC_ACT_OK;
    
    if (iph->daddr != 0x08080808)
        return TC_ACT_OK;
    
    bpf_printk("Blocking ICMP to 8.8.8.8");
    return TC_ACT_SHOT;
}

char _license[] SEC("license") = "GPL";
