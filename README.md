# mmd-auto-trace-4

## 環境構築

### CUDA

```
(base) miu@garnet:~$ nvcc --version
nvcc: NVIDIA (R) Cuda compiler driver
Copyright (c) 2005-2022 NVIDIA Corporation
Built on Wed_Jun__8_16:49:14_PDT_2022
Cuda compilation tools, release 11.7, V11.7.99
Build cuda_11.7.r11.7/compiler.31442593_0
```

### env

```
conda create --name mat4 python=3.10
conda activate mat4
conda install pytorch==2.0.1 torchvision==0.15.2 torchaudio==2.0.2 pytorch-cuda=11.7 -c pytorch -c nvidia

export PATH=/home/miu/anaconda3/envs/mat4/bin:$PATH
pip install -r requirements.txt
```

### データ配置

```
mmd-auto-trace-4/data/basicModel_neutral_lbs_10_207_0_v1.0.0.pkl
```



conda remove -n mat4 --all







