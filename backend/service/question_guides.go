package service

type QuestionGuidesService struct {
}

func (q *QuestionGuidesService) GetQuestionGuides() (string, error) {
	return `<h1>빅카인즈 AI - 이용자 질문 가이드</h1>
    <ol>
        <li>
            <span class="bold">원하는 내용을 <u>정확하게 검색</u>을 하려면 <em>질문의 의도</em>를 정확히 알려주세요. </span>
                <ul>
                    <li>(X) 정부의 정책에 대해서 자세히 알려줄래?</li>
                    <li>(O) 정부의 <em>부동산 정책</em>에 대해서 자세히 알려줄래?</li>
                    <li>(X) 현 정부는 언제부터 언제까지야?</li>
                    <li>(O) 현 정부의 <em>임기</em>는 언제 시작해서 언제 끝나?</li>
                </ul>
        </li>
        <li>
            <span class="bold"><u>특정 기간의 기사를 검색</u>하고 싶다면 <em>정확한 기간</em>을 알려주세요.</span>
                <ul>
                    <li><em>최근</em>이라는 표현을 사용하면 <u>한 달 이내</u>의 기사를 더 쉽게 찾을 수 있습니다.
                        <ul>
                            <li>(X) 정부의 출산율 정책에 대해 알려줄래?</li>
                            <li>(O) <em>최근</em> 정부의 출산율 정책에 대해 알려줄래?</li>
                        </ul>
                    </li>
                    <li>
                        <em>특정 기간</em>을 지정하면 "<u>해당 날짜</u>"에 해당하는 기사로 검색해 드립니다.
                        <ul>
                            <li>(X) 손흥민 선수 관련 기사를 찾아줄래?</li>
                            <li>(O) <em>2023년 11월 20일부터 최근까지</em>의 손흥민 선수 관련 기사를 찾아줄래?</li>
                            <li>(X) 한국시리즈 우승팀은 어디야? </li>
                            <li>(O) <em>올해</em> 한국시리즈 우승팀은 어디야?</li>
                        </ul>
                    </li>
                </ul>
            </li>
            <li>
                <span class="bold">요약/ 번역/ 맞춤법 검사를 도와 드립니다.</span>
                <ul>
                    <li>
                        <u>요약이 필요할 때</u>는 <em>요약</em>해 달라고 요청하세요.
                        <ul>
                            <li>요약이 필요한 글을 입력한 후, 아래에 “위 내용을 <em>요약해줄래</em>?” 라고 입력하세요.</li>
                        </ul>
                    </li>
                    <li><u>번역이 필요할 때</u>는 <em>번역</em>해 달라고 요청하세요.
                        <ul>
                            <li>번역이 필요한 글을 입력한 후, 아래에 “위 내용을 <em>번역해줄래</em>?” 라고 입력하세요.</li>
                        </ul>
                    </li>
                    <li><u>맞춤법 검사가 필요할 때</u>는 <em>맞춤법 검사</em>해 달라고 요청하세요.
                        <ul>
                            <li>맞춤법 검사가 필요한 글을 입력한 후, 아래에 “위 내용을 <em>맞춤법 검사해줄래</em>?” 라고 입력하세요.</li>
                        </ul>
                    </li>
                </ul>
            </li>
    </ol>`, nil
}

func (q *QuestionGuidesService) GetQuestionGuidesTips() ([]string, error) {
	return []string{
		"원하는 내용을 정확하게 검색하려면 '질문의 의도'를 명확히 알려주세요.",
		"'최근'이라는 표현을 사용하면 한 달 이내의 기사를 더 쉽게 찾을 수 있습니다.",
		"'특정 기간'을 지정하면 해당 날짜에 해당하는 기사로 검색해 드립니다.",
		"'~에 대해 분석해줘'와 같은 질문을 통해 보다 자세한 답변을 받을 수 있습니다.",
		"요약이 필요하다면 <문서 내용>을 하나의 입력창에 먼저 입력하고 '위 내용을 요약해 줄래?'라고 요청하세요.",
		"번역이 필요할 때는 <문서 내용>을 먼저 입력한 후 '위 내용을 번역해 줄래?'라고 요청하세요.",
		"맞춤법 검사를 원하실 때는 <문서 내용>을 먼저 입력하고 '위 내용을 맞춤법 검사해줄래?'라고 요청하세요.",
		"답변 내용을 확인하실 때는 문장 옆이나 하단에 표시된 출처를 반드시 확인하세요.",
		"부적절한 내용이나 개인정보는 절대 입력하지 마세요.",
		"질문이 너무 길어질 경우 답변의 품질이 저하될 수 있습니다.",
	}, nil
}
