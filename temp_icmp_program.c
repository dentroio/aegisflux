#include <linux/bpf.h>
#include <linux/ip.h>
#include <linux/icmp.h>

SEC("tc")
int block_icmp_egress(struct __sk_buff *skb) {
    // Parse packet headers
    void *data = (void *)(long)skb->data;
    void *data_end = (void *)(long)skb->data_end;
    
    // Check packet bounds
    if (data + sizeof(struct iphdr) > data_end)
        return TC_ACT_OK;
    
    struct iphdr *iph = (struct iphdr *)data;
    
    // Check if IPv4 packet
    if (iph->version != 4)
        return TC_ACT_OK;
    
    // Check if ICMP packet
    if (iph->protocol != IPPROTO_ICMP)
        return TC_ACT_OK;
    
    // Check destination IP (8.8.8.8 = 0x08080808 in network byte order)
    if (iph->daddr != 0x08080808)
        return TC_ACT_OK;
    
    // Block this ICMP packet to 8.8.8.8
    bpf_printk("Blocking ICMP packet to 8.8.8.8");
    return TC_ACT_SHOT;  // Drop packet
}

char _license[] SEC("license") = "GPL";
