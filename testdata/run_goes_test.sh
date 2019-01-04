#!/bin/bash

if ! [ $(id -u) = 0 ]; then
   echo "You must run this script as root/sudo."
   exit 1
fi

# bb must run from parent directory to locate testdata/netport.yaml
cd ..
tester=./goes-platina-mk1-blackbox.test

log_dir="testdata/log"
mkdir -p ${log_dir}
echo "log directory: "${log_dir}

touch ${log_dir}/test.log
date >> ${log_dir}/test.log


testcases=(
    "Test/net/ping"
    "Test/net/dhcp"
    "Test/net/static"
    "Test/net/gobgp"
    "Test/net/bird/bgp"
    "Test/net/bird/ospf"
    "Test/net/frr/bgp"
    "Test/net/frr/ospf"
    "Test/net/frr/isis"
    "Test/vlan/ping"
    "Test/vlan/dhcp"
    "Test/vlan/slice"
    "Test/vlan/static"
    "Test/vlan/gobgp"
    "Test/vlan/bird/bgp"
    "Test/vlan/bird/ospf"
    "Test/vlan/frr/bgp"
    "Test/vlan/frr/ospf"
    "Test/vlan/frr/isis"
    "Test/bridge/ping"
    "Test/nsif"
    "Test/multipath"
)

quit=0
fails=0

sigint() {
    echo "quit after current testcase"
    quit=1
}

trap 'sigint'  INT

fix_it() {
  goes stop
  docker stop R1 R2 R3 R4 > /dev/null 2>&1
  docker rm -v R1 R2 R3 R4 > /dev/null 2>&1
  docker stop CA-1 RA-1 RA-2 CA-2 CB-1 RB-1 RB-2 CB-2 > /dev/null 2>&1
  docker rm -v CA-1 RA-1 RA-2 CA-2 CB-1 RB-1 RB-2 CB-2 > /dev/null 2>&1
  ip -all netns del
  ./testdata/xeth_util.sh test_init
}


if [ "$1" == "fix_it" ]; then
    fix_it
elif [ "$1" == "dryrun" ]; then
    ${tester} -test.v  -test.dryrun
elif [ "$1" == "list" ]; then
    id=0
    for t in ${testcases[@]}; do
        id=$(($id+1))	
        echo $id ":" $t
    done
elif [ "$1" == "run" ]; then
    test_range=${testcases[@]}
elif [ "$1" == "run_range" ]; then
    shift
    start=$1
    start=$(($start-1))
    shift
    stop=$1
    stop=$(($stop-1))
    shift
    if [ "$1" == "verbose" ]; then
       test_flags="-test.vvv"
       shift
    fi
    test_range=""
    for i in $(seq $start $stop); do
        test_range="${test_range} ${testcases[$i]}"
    done
elif [ "$1" == "run_step" ]; then
    shift
    start=$1
    start=$(($start-1))
    echo "call test directly..."
    echo "(cd ..; ${tester} -test.vvv -test.pause -test.step -test.run=${testcases[$start]})"
    exit
else
    echo "fix_it | dryrun | list | run | run_range <start end [verbose]> | run_step <test_num>"
fi

test_count=$(echo $test_range | wc -w)

if [ $test_count != 0 ]; then
    echo "Running $test_count tests"
else
    exit 0
fi

if [ -z "$GOPATH" ]; then
    echo "GOPATH not set, try 'sudo -E ./$0 $*'"
    exit 1
fi

count=0

fix_it

for t in ${test_range}; do
    log=${t//\//_}.out
    count=$(($count+1))
    printf "Running %46s " $t" ($count of $test_count) : "
    echo "$GOPATH ${tester} -test.v ${test_flags} -test.run=$t " >> ${log_dir}/test.log
    GOPATH=$GOPATH ${tester} -test.v ${test_flags} -test.run=$t > ${log_dir}/test.out 2>&1

    if [ $? == 0 ]; then
        echo "OK"
        mv ${log_dir}/test.out ${log_dir}/$log.OK
    else
        mv ${log_dir}/test.out ${log_dir}/$log.error
        if grep -q panic ${log_dir}/$log.error; then
            echo "Crashed"
        else
            echo "Failed"
        fi
	fails=$(($fails+1))
    fi

    if [ "$quit" -eq 1 ]; then
        echo "Aborted"
        break
    fi
done
echo
echo "$fails testcase(s) failed."

echo fix_it to cleanup

exit 0
