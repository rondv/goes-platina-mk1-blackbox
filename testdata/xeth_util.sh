#!/bin/bash

xeth_driver="platina-mk1"
xeth_type="xeth"

# in order: xeth, vlan, br
xeth_fp()
{
    for i in $(ls -1 /sys/class/net); do
        if [ $i == "lo" ]; then
            continue
        fi
        if $(ethtool -i $i | egrep -q -e "driver: xeth"); then
            echo $i
        fi
    done
}
xeth_vlan()
{
    for i in $(ls -1 /sys/class/net); do
        if [ $i == "lo" ]; then
            continue
        fi
        if $(ethtool -i $i | egrep -q -e VLAN); then
            echo $i
        fi
    done
}

xeth_br()
{
    for i in $(ls -1 /sys/class/net); do
        if [ $i == "lo" ]; then
            continue
        fi
        if $(ethtool -i $i | egrep -q -e "driver: bridge"); then
            echo $i
        fi
    done
}

xeth_range()
{
    start=$1
    shift
    stop=$1
    shift

    for i in $(seq $start $stop); do
        echo -n xeth$i" "
    done
}

eth_range()
{
    start=$1
    shift
    stop=$1
    shift

    for i in $(seq $start $stop); do
        echo -n eth-$i-0" "
    done
}

xeth_up()
{
    for i in $xeth_list; do
        ip link set $i up
    done
}

xeth_down()
{
    for i in $xeth_list; do
        ip link set $i down
    done
}

xeth_carrier()
{
    for i in $xeth_list; do
        goes ip link set $i +carrier
    done
}

xeth_no_carrier()
{
    for i in $xeth_list; do
        goes ip link set $i -carrier
    done
}

xeth_flap()
{
    xeth_down $xeth_list
    xeth_up $xeth_list
}

xeth_add()
{
    for i in $xeth_list; do
        ip link add $i type ${xeth_type}
        ip link set $i up
        ethtool -s $i speed 100000 autoneg off
    done
}

xeth_netport_add()
{
    xeth_list=$(grep -o " .eth.*" netport.yaml)
    xeth_add
    xeth_flap $xeth_list
}

xeth_del()
{
    for i in $xeth_list; do
        ip link del $i
    done
}

xeth_show()
{
    for i in $xeth_list; do
        #ip link show $i
        ip addr show dev $i
    done
}

xeth_br_show()
{
    for i in $(xeth_br); do
        echo
        echo "bridge "$i
        bridge link | grep $i
        bridge fdb | grep $i
    done
}

xeth_echo()
{
    for i in $xeth_list; do
        echo -n $i" "
    done
    echo
}

xeth_isup()
{
    xeth_show $xeth_list | grep -i state.up | wc -l
}

xeth_stat()
{
    for i in $xeth_list; do
        echo $i
        ethtool -S $i
    done
}

xeth_to_netns()
{
    netns=$1
    shift

    for i in $xeth_list; do
        ip link set $i netns $netns
    done
    ip netns exec $netns ./xeth_util.sh flap
    ip netns exec $netns ./xeth_util.sh show
}

xeth_netns_list()
{
    echo $(ip netns | sed -e "s/ .*//" | sort -V)
}

xeth_netns_del()
{
    for netns in $(xeth_netns_list); do
      for i in $(ip netns exec $netns ./xeth_util.sh echo); do
        ip netns exec $netns ip link set $i netns 1
      done
      ip netns del $netns
    done
}

xeth_netns_show()
{
    show_arp=false
    show_ip=false
    show_route=false
    show_vrf=false

    if [ "$1" == "arp" ]; then
      show_arp=true
      shift
    fi
    if [ "$1" == "ip" ]; then
      show_ip=true
      shift
    fi
    if [ "$1" == "route" ]; then
      show_route=true
      shift
    fi
    if [ "$1" == "vrf" ]; then
      show_vrf=true
      shift
    fi

    echo "netns default"
    if $show_arp; then
      arp
    fi
    if $show_ip; then
      ./xeth_util.sh show | grep -e 'inet '|sed -e "s/inet \(.*\) scope global \(.*\)/\2\t\1/"
      ./xeth_util.sh br   | grep -e 'inet '|sed -e "s/inet \(.*\) scope global \(.*\)/\2\t\1/"
    else
      ./xeth_util.sh show
      ./xeth_util.sh br
    fi
    if $show_route; then
      ip route
    fi

    for netns in $(xeth_netns_list); do
      echo
      echo "netns "$netns
      if $show_arp; then
        arp
      fi
      if $show_ip; then
        ip netns exec $netns ./xeth_util.sh show | grep -e 'inet '|sed -e "s/inet \(.*\) scope global \(.*\)/\2\t\1/"
        ip netns exec $netns ./xeth_util.sh br   | grep -e 'inet '|sed -e "s/inet \(.*\) scope global \(.*\)/\2\t\1/"
      else
        ip netns exec $netns ./xeth_util.sh show
        ip netns exec $netns ./xeth_util.sh br
      fi
      if $show_route; then
        ip netns exec $netns ip route
      fi
    done
    if $show_vrf; then
      echo ---
      echo vrf
      echo ---
      goes vnet show fe1 l3 |grep -o " intf.*vrf[^ ]* " |sort -uV|sed -e "s/intf.//; s/vrf.//; s/\/.*//" |grep -v 0$
    fi
}

xeth_netns_echo()
{
    for netns in $(xeth_netns_list); do
      echo -n "netns "$netns": "
      ip netns exec $netns ./xeth_util.sh echo
    done
}

xeth_netns_flap()
{
    ./xeth_util.sh flap
    for netns in $(xeth_netns_list); do
      echo "netns "$netns
      ip netns exec $netns ./xeth_util.sh flap
    done
}

xeth_netns_carrier()
{
    for netns in $(xeth_netns_list); do
      echo "netns "$netns
      ip netns exec $netns ./xeth_util.sh carrier
    done
}

range="all"
if [ $# -gt 0 ]; then
    range=$1
fi

if [ $range == "xeth_range" ]; then
    shift
    start=$1
    shift
    stop=$1
    shift
    xeth_list=$(xeth_range $start $stop)

elif [ $range == "eth_range" ]; then
    shift
    start=$1
    shift
    stop=$1
    shift
    xeth_list=$(eth_range $start $stop)

else
    xeth_list="$(xeth_fp | sort -V)" 
    xeth_list+=" $(xeth_vlan | sort -V)"
    xeth_list+=" $(xeth_br | sort -V)"
fi

cmd="help"
if [ $# -gt 0 ]; then
    cmd=$1
    shift
fi

# echo range = $xeth_list
# echo command = $cmd

if [ $cmd == "show" ]; then
    xeth_show $xeth_list
elif [ $cmd == "br" ]; then
    xeth_br_show
elif [ $cmd == "showup" ]; then
    xeth_show $xeth_list | grep -i state.up
elif [ $cmd == "echo" ]; then
    xeth_echo $xeth_list
elif [ $cmd == "reset" ]; then
    # FIXME: also remove vlan interfaces
    xeth_netns_del
    rmmod ${xeth_driver}
    modprobe ${xeth_driver}
elif [ $cmd == "test_init" ]; then
    rmmod ${xeth_driver}
    modprobe ${xeth_driver}
elif [ $cmd == "up" ]; then
    xeth_up $xeth_list
elif [ $cmd == "carrier" ]; then
    xeth_carrier $xeth_list
elif [ $cmd == "down" ]; then
    xeth_down $xeth_list
elif [ $cmd == "flap" ]; then
    xeth_flap $xeth_list
elif [ $cmd == "isup" ]; then
    xeth_isup
elif [ $cmd == "stat" ]; then
    xeth_stat $xeth_list | grep -v " 0$"
elif [ $cmd == "to_netns" ]; then
    xeth_to_netns $1 $xeth_list
elif [ $cmd == "netns_del" ]; then
    xeth_netns_del
elif [ $cmd == "netns_show" ]; then
    xeth_netns_show $*
elif [ $cmd == "netns_showup" ]; then
    xeth_netns_show $* | egrep -i -e netns -e state.up
elif [ $cmd == "netns_echo" ]; then
    echo "list: $(xeth_netns_list)"
    echo "default: "$(xeth_echo)
    xeth_netns_echo
elif [ $cmd == "netns_flap" ]; then
    xeth_netns_flap
elif [ $cmd == "netns_carrier" ]; then
    xeth_netns_carrier
else
    # help
    grep range.*[t]hen $0 | grep -o \".*\"
    grep cmd.*[t]hen $0 | grep -o \".*\"
fi
