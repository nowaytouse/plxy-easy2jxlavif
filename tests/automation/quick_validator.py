#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Pixly 快速验证脚本
专门检查质量判断引擎和格式转换的准确性
"""

import sys
import os
import subprocess
import json
import shutil
import time
from pathlib import Path
from datetime import datetime
from typing import Dict, List, Tuple

class PixlyQuickValidator:
    """Pixly快速验证器"""
    
    def __init__(self, pixly_binary: str, test_data_dir: str):
        self.pixly_binary = Path(pixly_binary)
        self.test_data_dir = Path(test_data_dir)
        self.timestamp = datetime.now().strftime('%Y%m%d_%H%M%S')
        
        # 创建验证工作区
        self.work_dir = Path(f"./pixly_validation_{self.timestamp}")
        self.work_dir.mkdir(exist_ok=True)
        
        print(f"Pixly快速验证器初始化完成")
        print(f"工作目录: {self.work_dir.absolute()}")
    
    def create_test_subset(self, subset_name: str, file_patterns: List[str]) -> Path:
        """创建测试文件子集"""
        subset_dir = self.work_dir / subset_name
        subset_dir.mkdir(exist_ok=True)
        
        copied_files = []
        for pattern in file_patterns:
            # 查找匹配的文件
            matching_files = list(self.test_data_dir.glob(f"*{pattern}*"))
            for file_path in matching_files[:2]:  # 每种类型最多复制2个文件
                if file_path.is_file():
                    dest_path = subset_dir / file_path.name
                    shutil.copy2(file_path, dest_path)
                    copied_files.append(dest_path.name)
                    print(f"  复制文件: {file_path.name}")
        
        print(f"创建测试子集 '{subset_name}': {len(copied_files)} 个文件")
        return subset_dir
    
    def run_quick_test(self, test_dir: Path, mode: str, timeout: int = 60) -> Dict:
        """运行快速测试"""
        print(f"\n运行快速测试: {test_dir.name} (模式: {mode})")
        
        # 准备输入
        inputs = f"{test_dir.absolute()}\n{mode}\n"
        
        try:
            # 运行Pixly
            start_time = time.time()
            process = subprocess.Popen(
                [str(self.pixly_binary)],
                stdin=subprocess.PIPE,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True,
                cwd=self.pixly_binary.parent
            )
            
            stdout, stderr = process.communicate(input=inputs, timeout=timeout)
            end_time = time.time()
            
            return {
                'success': process.returncode == 0,
                'duration': end_time - start_time,
                'stdout': stdout,
                'stderr': stderr,
                'return_code': process.returncode
            }
            
        except subprocess.TimeoutExpired:
            process.kill()
            return {
                'success': False,
                'duration': timeout,
                'stdout': '',
                'stderr': 'Timeout',
                'return_code': -1
            }
        except Exception as e:
            return {
                'success': False,
                'duration': 0,
                'stdout': '',
                'stderr': str(e),
                'return_code': -2
            }
    
    def analyze_quality_engine_output(self, stdout: str) -> Dict:
        """分析质量判断引擎输出"""
        analysis = {
            'quality_engine_working': True,
            'issues': [],
            'detected_qualities': [],
            'corrupted_files': 0,
            'processed_files': 0
        }
        
        lines = stdout.split('\n')
        
        for line in lines:
            line_lower = line.lower()
            
            # 检查质量判断引擎是否正常工作
            if '质量评估' in line or 'quality assessment' in line_lower:
                analysis['quality_engine_working'] = True
            
            # 检查误报问题
            if '全部损坏' in line or '100%损坏' in line:
                analysis['issues'].append("质量判断引擎误报所有文件为损坏")
                analysis['quality_engine_working'] = False
            
            if '误报' in line:
                analysis['issues'].append(f"发现误报提及: {line.strip()}")
            
            # 提取质量信息
            if '品质' in line or 'quality' in line_lower:
                if '高品质' in line or 'high quality' in line_lower:
                    analysis['detected_qualities'].append('high')
                elif '低品质' in line or 'low quality' in line_lower:
                    analysis['detected_qualities'].append('low')
                elif '中品质' in line or 'medium quality' in line_lower:
                    analysis['detected_qualities'].append('medium')
            
            # 统计处理的文件
            if '检测到' in line and '损坏' in line:
                try:
                    # 尝试提取数字
                    import re
                    numbers = re.findall(r'\d+', line)
                    if numbers:
                        analysis['corrupted_files'] = int(numbers[0])
                except:
                    pass
            
            if '个文件' in line or 'files' in line_lower:
                try:
                    import re
                    numbers = re.findall(r'\d+', line)
                    if numbers:
                        analysis['processed_files'] = max(analysis['processed_files'], int(numbers[0]))
                except:
                    pass
        
        return analysis
    
    def check_format_conversion(self, test_dir: Path, expected_formats: List[str]) -> Dict:
        """检查格式转换结果"""
        result = {
            'conversions_found': [],
            'missing_conversions': [],
            'unexpected_files': [],
            'success': True
        }
        
        # 扫描转换后的文件
        current_files = {}
        for file_path in test_dir.iterdir():
            if file_path.is_file() and not file_path.name.startswith('.'):
                suffix = file_path.suffix.lower()
                if suffix not in current_files:
                    current_files[suffix] = []
                current_files[suffix].append(file_path.name)
        
        # 检查预期格式
        for expected in expected_formats:
            if expected in current_files:
                result['conversions_found'].append(expected)
            else:
                result['missing_conversions'].append(expected)
                result['success'] = False
        
        # 检查意外文件
        suspicious_extensions = ['.tmp', '.temp', '.bak', '.backup', '.pixly_']
        for ext in suspicious_extensions:
            if ext in current_files:
                result['unexpected_files'].extend(current_files[ext])
        
        return result
    
    def run_comprehensive_validation(self):
        """运行综合验证"""
        print("开始Pixly综合验证...")
        
        validation_results = {
            'timestamp': self.timestamp,
            'tests': [],
            'summary': {
                'total': 0,
                'passed': 0,
                'failed': 0,
                'quality_engine_issues': [],
                'conversion_issues': []
            }
        }
        
        # 定义测试场景
        test_scenarios = [
            {
                'name': 'basic_images',
                'file_patterns': ['.jpg', '.png', '.webp'],
                'mode': '1',  # 自动模式+
                'expected_formats': ['.jxl', '.avif'],
                'timeout': 30
            },
            {
                'name': 'high_quality_images',
                'file_patterns': ['.tiff', '.heif'],
                'mode': '2',  # 品质模式
                'expected_formats': ['.jxl'],
                'timeout': 45
            },
            {
                'name': 'mixed_media',
                'file_patterns': ['.gif', '.mp4', '.mov'],
                'mode': '1',  # 自动模式+
                'expected_formats': ['.jxl', '.avif'],
                'timeout': 60
            },
            {
                'name': 'sticker_mode',
                'file_patterns': ['.png', '.jpg'],
                'mode': '3',  # 表情包模式
                'expected_formats': ['.avif'],
                'timeout': 30
            }
        ]
        
        for scenario in test_scenarios:
            print(f"\n{'='*50}")
            print(f"测试场景: {scenario['name']}")
            print(f"{'='*50}")
            
            # 创建测试子集
            test_dir = self.create_test_subset(scenario['name'], scenario['file_patterns'])
            
            # 运行测试
            test_result = self.run_quick_test(test_dir, scenario['mode'], scenario['timeout'])
            
            # 分析质量判断引擎
            quality_analysis = self.analyze_quality_engine_output(test_result['stdout'])
            
            # 检查格式转换
            format_check = self.check_format_conversion(test_dir, scenario['expected_formats'])
            
            # 整合结果
            scenario_result = {
                'name': scenario['name'],
                'success': test_result['success'] and quality_analysis['quality_engine_working'] and format_check['success'],
                'duration': test_result['duration'],
                'quality_analysis': quality_analysis,
                'format_check': format_check,
                'issues': []
            }
            
            # 收集问题
            if not test_result['success']:
                scenario_result['issues'].append(f"进程执行失败: 返回码 {test_result['return_code']}")
            
            if test_result['stderr']:
                scenario_result['issues'].append(f"标准错误: {test_result['stderr'][:100]}...")
            
            scenario_result['issues'].extend(quality_analysis['issues'])
            
            if format_check['missing_conversions']:
                scenario_result['issues'].append(f"缺失预期格式: {format_check['missing_conversions']}")
            
            if format_check['unexpected_files']:
                scenario_result['issues'].append(f"发现意外文件: {format_check['unexpected_files']}")
            
            # 输出结果
            status = "✅ 通过" if scenario_result['success'] else "❌ 失败"
            print(f"结果: {status} (耗时: {scenario_result['duration']:.1f}s)")
            
            if scenario_result['issues']:
                print("发现问题:")
                for issue in scenario_result['issues']:
                    print(f"  - {issue}")
            
            if quality_analysis['detected_qualities']:
                print(f"检测到的质量级别: {set(quality_analysis['detected_qualities'])}")
            
            if format_check['conversions_found']:
                print(f"成功转换为: {format_check['conversions_found']}")
            
            validation_results['tests'].append(scenario_result)
        
        # 生成汇总
        validation_results['summary']['total'] = len(validation_results['tests'])
        validation_results['summary']['passed'] = len([t for t in validation_results['tests'] if t['success']])
        validation_results['summary']['failed'] = validation_results['summary']['total'] - validation_results['summary']['passed']
        
        # 收集质量引擎和转换问题
        for test in validation_results['tests']:
            for issue in test['issues']:
                if '质量判断' in issue or '误报' in issue:
                    validation_results['summary']['quality_engine_issues'].append(issue)
                elif '格式' in issue or '转换' in issue:
                    validation_results['summary']['conversion_issues'].append(issue)
        
        # 保存详细结果
        results_file = self.work_dir / f"validation_results_{self.timestamp}.json"
        with open(results_file, 'w', encoding='utf-8') as f:
            json.dump(validation_results, f, ensure_ascii=False, indent=2)
        
        # 输出最终汇总
        print(f"\n{'='*60}")
        print("验证汇总")
        print(f"{'='*60}")
        print(f"总测试: {validation_results['summary']['total']}")
        print(f"通过: {validation_results['summary']['passed']}")
        print(f"失败: {validation_results['summary']['failed']}")
        print(f"成功率: {validation_results['summary']['passed']/validation_results['summary']['total']*100:.1f}%")
        
        if validation_results['summary']['quality_engine_issues']:
            print(f"\n质量判断引擎问题 ({len(validation_results['summary']['quality_engine_issues'])}):")
            for issue in validation_results['summary']['quality_engine_issues']:
                print(f"  - {issue}")
        
        if validation_results['summary']['conversion_issues']:
            print(f"\n格式转换问题 ({len(validation_results['summary']['conversion_issues'])}):")
            for issue in validation_results['summary']['conversion_issues']:
                print(f"  - {issue}")
        
        print(f"\n详细结果保存至: {results_file}")
        
        return validation_results


def main():
    """主函数"""
    # 配置路径
    pixly_binary = "/Users/nameko_1/Documents/Pixly/Go_Source_code_Updata/pixly_test"
    test_data_dir = "/Users/nameko_1/Documents/Pixly/test_pack_all/不同格式测试合集_测试运行"
    
    # 检查路径
    if not Path(pixly_binary).exists():
        print(f"错误: Pixly二进制文件不存在: {pixly_binary}")
        print("请先构建项目: go build -o pixly_test .")
        return 1
    
    if not Path(test_data_dir).exists():
        print(f"错误: 测试数据目录不存在: {test_data_dir}")
        return 1
    
    # 运行验证
    validator = PixlyQuickValidator(pixly_binary, test_data_dir)
    results = validator.run_comprehensive_validation()
    
    # 返回状态码
    return 0 if results['summary']['failed'] == 0 else 1


if __name__ == "__main__":
    sys.exit(main())