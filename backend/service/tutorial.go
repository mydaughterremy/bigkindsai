package service

type TutorialService struct {
}

func (t *TutorialService) GetTutorial() (string, error) {
	return `<ol class="guide_list">
    <li>
        <span class="bold">질문의 의도를 포함해 <em>구체적으로 정확히 질문</em>하세요.</span>                            
    </li>
    <li>
        <span class="bold"><em>'최근'을 사용</em>하면 한 달 이내의 기사를 더 쉽게 찾을 수 있습니다.</span>                            
    </li>
    <li>
        <span class="bold"><em>‘특정기간’을 지정</em>하면 ‘해당 날짜’의 기사를 검색해 드립니다.</span>                            
    </li>
    <li>
        <span class="bold"><em>‘~에 대해 분석해줘’</em>와 같이 질문하면 보다 자세한 답변을 제공합니다.</span>                            
    </li>
    <li>
        <span class="bold">질문이 <em>너무 길어질 경우 답변의 품질이 저하</em>될 수 있습니다.</span>                            
    </li>
    <li>
        <span class="bold">답변 내용은 함께 표시되는 <em>출처를 반드시 확인</em>하세요.</span>                            
    </li>
    <li>
        <span class="bold">요약 / 번역 / 맞춤법 검사가 필요할 때 <em>&lt;문서내용&gt;을 하나의 입력창에 먼저 넣고</em>
            - ‘위 내용 요약 / 번역 / 맞춤법 검사 해 줘’라고 요청하세요.
       </span>                            
    </li>
    <li>
        <span class="bold">부적절한 내용이나 개인정보는 절대 입력하지 마세요.</span>                            
    </li>
</ol>`, nil
}
