project_path=$(cd `dirname $0`;pwd)
project_name="${project_path##*/}"
go_id=`ps -ef|grep "./$project_name" |grep -v "grep" | awk '{print $2}'` 
if [ -z "$go_id" ];
then
 echo "[go pid not found]"
else
 kill -9 $go_id
 echo "killed $go_id"
fi

echo "clean old file"
rm -rf $project_name
if [ -f build ]; then
 echo "strat new process"
 cp ./build ./$project_name
 chmod -R 777 $project_name
 nowDate=$(date +%F)
 nohup ./$project_name 2>&1 | cronolog ./logs/%Y-%m-%d.log &
else
 echo "app file not found,qiut"
fi
